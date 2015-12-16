package main_test

import (
	"net"
	"log"
	"io/ioutil"
	. "github.com/hkparker/sit"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/gbytes"
	"os"
	"os/exec"
	"fmt"
	"crypto/tls"
)

func CreateConnection(service string) (client, server net.Conn) {
	socket := make(chan net.Conn, 1)
	listener, err := net.Listen("tcp", service)
	Expect(err).To(BeNil())
	go func() {
		conn, _ := listener.Accept()
		socket <- conn
	}()
	client, err = net.Dial("tcp", service)
	Expect(err).To(BeNil())
	server = <-socket
	return client, server
}

var _ = Describe("Sit", func() {
	log.SetOutput(ioutil.Discard)

	var binary_path string
	BeforeSuite(func() {
		var err error
		binary_path, err = gexec.Build("github.com/hkparker/sit")
		Expect(err).To(BeNil())
	})

	AfterSuite(func() {
	    gexec.CleanupBuildArtifacts()
	})

	Describe("ExchangeData", func() {

		It("copies data in both directions", func() {
			client, server := CreateConnection("127.0.0.1:1234")
			defer client.Close()
			defer server.Close()
			go ExchangeData(server, client)
			down_msg := "Going down"
			up_msg := "Going up"
			server.Write([]byte(down_msg))
			client.Write([]byte(up_msg))
			server_msg := make([]byte, 10)
			n, err := client.Read(server_msg)
			Expect(err).To(BeNil())
			Expect(n).To(Equal(10))
			Expect(string(server_msg)).To(Equal(down_msg))
			client_msg := make([]byte, 8)
			n, err = server.Read(client_msg)
			Expect(err).To(BeNil())
			Expect(n).To(Equal(8))
			Expect(string(client_msg)).To(Equal(up_msg))
		})

		It("closes both sockets and returns when one closes", func() {
			client, server := CreateConnection("127.0.0.1:2345")
			defer client.Close()
			defer server.Close()
			returned := make(chan bool, 1)
			go func() {
				ExchangeData(server, client)
				returned <- true
			}()
			client.Close()
			server.Write([]byte("closed"))
			Expect(<-returned).To(Equal(true))
			err := server.Close()
			Expect(err).ToNot(BeNil())
		})
	})

	Describe("ProxyBack", func() {
		It("dials the backend service when frontend connection received", func() {
			backend := "127.0.0.1:3456"
			listener, err := net.Listen("tcp", backend)
			defer listener.Close()
			Expect(err).To(BeNil())
			connected := make(chan bool)
			go func() {
				_, err := listener.Accept()
				Expect(err).To(BeNil())
				connected <- true
			}()
			client, _ := CreateConnection("127.0.0.1:3457")
			go ProxyBack(client, backend)
			Expect(<-connected).To(BeTrue())
		})

		It("closes the frontend connection when it can't dial the backend service", func() {
			client, _ := CreateConnection("127.0.0.1:4567")
			ProxyBack(client, "127.0.0.1:4568")
			err := client.Close()
			Expect(err).ToNot(BeNil())
		})
	})

	Describe("main", func() {

		It("listens and connects to services defined in environment", func() {
			accepted := make(chan bool)
			backend, err := net.Listen("tcp", "127.0.0.1:5678")
			Expect(err).To(BeNil())
			go func() {
				conn, err := backend.Accept()
				accepted <- true
				Expect(err).To(BeNil())
				data := make([]byte, 14)
				n, err := conn.Read(data)
				Expect(err).To(BeNil())
				Expect(n).To(Equal(14))
				Expect(string(data)).To(Equal("frontend hello"))
				conn.Write([]byte("backend hello"))
			}()

			test_cert := `-----BEGIN CERTIFICATE-----
MIIC+TCCAeGgAwIBAgIQLd80irj1mqXj5/nAVRVW9TANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMB4XDTE1MTEwMzA2NDcwOFoXDTE2MTEwMjA2NDcw
OFowEjEQMA4GA1UEChMHQWNtZSBDbzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCC
AQoCggEBANohMLecF1Gl6DvhW2ZqiesBUDLPt/HBQY61GBb6TFedJFaae3OWnCtU
ni4Lw/47gfrbyFBygXS5Su6tbTcGqjhG5sXXDDUljDcf3J4MZmTY/ySr/IvhAG22
qZekANq4VVoAgWD8Z1CrjUX0NyTlYhWrnhH/sYhdo1PDe6ZpdsF19B+0UweSDR6U
B5c24P5JFMrearz2tO2PuxdZ+7LLkhRcMxot+dH+bXWrI6S0XCdIL45u5ZsLaDhx
Gmu6aYz65mZR0eIoxtqIk8Y5U6cEBGWNuRn2W/UArit7sXIzGaU6lwgkt/+3h9gh
IbGt0+1cYztztkJ/USBavynLY3yQNuUCAwEAAaNLMEkwDgYDVR0PAQH/BAQDAgWg
MBMGA1UdJQQMMAoGCCsGAQUFBwMBMAwGA1UdEwEB/wQCMAAwFAYDVR0RBA0wC4IJ
bG9jYWxob3N0MA0GCSqGSIb3DQEBCwUAA4IBAQB+NGxiZoWIuIB0djg0LOoqQKKs
/NrNdGD8sSYdtysbOHYYDuEQ3mRh49aGtXh8KGE/x7ovPHqUMx7cizICLgUxilgj
bOTdpIDWqmvNM9+QcMxVHjN5ydXDpQosktBvCMa09m1zRMaC6u/HFwGAu5JMwNI2
WWNz9YtkYhyAsPpcZoL6xs5ENs6m/7ERnIMq4vJgvLzT5vUXMgS3Thpoa62xb+wn
QF1qMqh36GEAyp9l4IV4bR/LAq5AcU2y1Xyx7inSU6/NYuFKJV270HY17NNdxt1p
PXBXOtTCTfGR4hREUsE0/1rHm2Tp/UErOSwvi6P5KZa1jaAXga6Ts3gaNAxV
-----END CERTIFICATE-----`
			test_key := `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA2iEwt5wXUaXoO+FbZmqJ6wFQMs+38cFBjrUYFvpMV50kVpp7
c5acK1SeLgvD/juB+tvIUHKBdLlK7q1tNwaqOEbmxdcMNSWMNx/cngxmZNj/JKv8
i+EAbbapl6QA2rhVWgCBYPxnUKuNRfQ3JOViFaueEf+xiF2jU8N7pml2wXX0H7RT
B5INHpQHlzbg/kkUyt5qvPa07Y+7F1n7ssuSFFwzGi350f5tdasjpLRcJ0gvjm7l
mwtoOHEaa7ppjPrmZlHR4ijG2oiTxjlTpwQEZY25GfZb9QCuK3uxcjMZpTqXCCS3
/7eH2CEhsa3T7VxjO3O2Qn9RIFq/KctjfJA25QIDAQABAoIBABcFq8OlSNzIOvOe
/GuZ0Qaps9I5FDwC3q12NZ2vO0GAB2bQlTkE62SzYKKRgVUi6LwWWFUNUNaF+/+m
9ED7hhm8efzljpdJaDwlM77GpkP8lylCsbv9unLZ9ZpYp/JtxCxko0MeQLVF9fot
JWjSgQCyhVN2/kDbSTK9Dh7pQKx4WbLNUD0wSdEOetKdgWLc82K9yLGN7JNRSvvc
yoGUO4l3cchtQeTv4YOiYZ322LxcExpR710O5jg59Q5ZgUDy54fFsb+EYf97uYz0
7MN6I3Puj2IMmbc62OEDne1aMT1iORAgCRmTrbof4AdwzmgarhD6qrNvK+yGq3EX
YE1HCj0CgYEA6z+giXHpRMEHdqyLM+nbEJmKLJMkOlkhoFiK31veZEBFKvjeLYrg
KgyCKbzRok5GfJOuPt2f/HHzvwwLaIT1iOnHGvaV25WbHiajYrmRNoiDzLrGIubj
5LEEU7bqguyarVx8voS47LPyRYuqNrgWzJn8oqFLOBzsnjrR03Xio08CgYEA7V78
LGIolDSH3shjD9i4P/BLtuDaIpjzQZmyMSmitcc/gwJLrY1VwxO+hn646lxKAAUw
I1EUshcXa5Kpuer2SMXQWT1JqBm/MgR1RXsqFOIuxh4f++uikzn53dNDA+rLH829
LE9UpMVzlRO8tUSgvQXiRS7FwcEfci4MDZ1fBYsCgYAzyVbyytO6IfAdrNAcBoAG
AHbNZzrTaWmgnb08fEHRueBAHHb0eZztRMGmpH1ViHu10uDJ0An3DbLFvMYKJTLU
B/qfsea9Zwq1sXXINueDpLu25urVJhTG9DzqnNq1JZbLUQ/Y9OismtRbgOpgj1fd
hIx71Jv1Z1CjaHlmXo4cuwKBgFe6HB7XJEIp6/E1RA9OPEI9L+5lyZixyG19PTMY
PS9LdTiY95kru/9945NXcEYFV8AMKs9SnwB2skwqhxnUMbORkD/6+6bp5RS6OxEz
xMi1Ey5bYdy8KHibG7KU6pafBvU5F2ox44mGBAKbqcmglHtnmkmRULscAeA0DnZV
rBGjAoGBALuvGYBey93z2gwlqej+kvU8G7Idq88DGmo5EkhRzNVUf7mqH6FIA0Ur
t+2WyoWKgpv5W2A7LEc1856znW4pdexvlveQcnIqxuZMSWRJdMa86pSQkLBqcCSb
g73r3HgKkqd2CJgEWShPz0JGMn9Caj9mbcQzcTjaRloVim7rOOUU
-----END RSA PRIVATE KEY-----`

			environment := os.Environ()
			environment = append(
				environment,
				fmt.Sprintf("CERT_PEM=%s", test_cert),
				fmt.Sprintf("CERT_KEY=%s", test_key),
				"FRONT_SERVICE=127.0.0.1:5679",
				"BACK_SERVICE=127.0.0.1:5678",
			)
			command := exec.Command(binary_path)
			command.Env = environment
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).To(BeNil())
			defer session.Terminate()

			Eventually(session.Out).Should(gbytes.Say("started listening for TLS connections on 127.0.0.1:5679"))
			config := &tls.Config{
				InsecureSkipVerify:	true,
			}
			frontend, err := tls.Dial("tcp", "127.0.0.1:5679", config)
			Expect(err).To(BeNil())
			Expect(<-accepted).To(Equal(true))

			frontend.Write([]byte("frontend hello"))
			data_back := make([]byte, 13)
			n, err := frontend.Read(data_back)
			Expect(n).To(Equal(13))
			Expect(err).To(BeNil())
			Expect(string(data_back)).To(Equal("backend hello"))
		})
	})
})
