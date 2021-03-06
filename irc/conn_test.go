package irc

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"log"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var ircd *mockIrcd

func init() {
	initTestServer()
}

func initTestServer() {
	ircd = &mockIrcd{
		conn:       make(chan bool, 1),
		connClosed: make(chan bool, 1),
	}
	ircd.start()
}

type mockIrcd struct {
	conn       chan bool
	connClosed chan bool
}

func (i *mockIrcd) start() {
	ln, err := net.Listen("tcp", "127.0.0.1:45678")
	if err != nil {
		log.Fatal(err)
	}

	cert, err := tls.X509KeyPair(testCert, testKey)
	if err != nil {
		log.Fatal(err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	lnTLS, err := tls.Listen("tcp", "127.0.0.1:45679", tlsConfig)
	if err != nil {
		log.Fatal(err)
	}

	go i.accept(ln)
	go i.accept(lnTLS)
}

func (i *mockIrcd) accept(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go i.handle(conn)
		i.conn <- true
	}
}

func (i *mockIrcd) handle(conn net.Conn) {
	buf := make([]byte, 1024)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			i.connClosed <- true
			return
		}
	}
}

func TestConnect(t *testing.T) {
	c := testClient()
	c.Connect("127.0.0.1:45678")
	assert.Equal(t, c.Host, "127.0.0.1")
	assert.Equal(t, c.Server, "127.0.0.1:45678")
	waitConnAndClose(t, c)
}

func TestConnectTLS(t *testing.T) {
	c := testClient()
	c.TLS = true
	c.TLSConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	c.Connect("127.0.0.1:45679")
	assert.Equal(t, c.Host, "127.0.0.1")
	assert.Equal(t, c.Server, "127.0.0.1:45679")
	waitConnAndClose(t, c)
}

func TestConnectDefaultPorts(t *testing.T) {
	c := testClient()
	c.Connect("127.0.0.1")
	assert.Equal(t, "127.0.0.1:6667", c.Server)

	c = testClient()
	c.TLS = true
	c.Connect("127.0.0.1")
	assert.Equal(t, "127.0.0.1:6697", c.Server)
}

func TestWrite(t *testing.T) {
	c, out := testClientSend()
	c.write("test")
	assert.Equal(t, "test\r\n", <-out)
	c.Write("test")
	assert.Equal(t, "test\r\n", <-out)
	c.writef("test %d", 2)
	assert.Equal(t, "test 2\r\n", <-out)
	c.Writef("test %d", 2)
	assert.Equal(t, "test 2\r\n", <-out)
}

func TestRecv(t *testing.T) {
	c := testClient()
	conn := &mockConn{hook: make(chan string, 16)}
	c.conn = conn

	buf := &bytes.Buffer{}
	buf.WriteString("CMD\r\n")
	buf.WriteString("PING :test\r\n")
	buf.WriteString("001\r\n")
	c.reader = bufio.NewReader(buf)

	c.ready.Add(1)
	c.sendRecv.Add(2)
	go c.send()
	go c.recv()

	assert.Equal(t, "PONG :test\r\n", <-conn.hook)
	assert.Equal(t, &Message{Command: "CMD"}, <-c.Messages)
}

func TestRecvTriggersReconnect(t *testing.T) {
	c := testClient()
	c.conn = &mockConn{}
	c.ready.Add(1)
	c.reader = bufio.NewReader(&bytes.Buffer{})
	done := make(chan struct{})
	ok := false
	go func() {
		c.sendRecv.Add(1)
		c.recv()
		_, ok = <-c.reconnect
		close(done)
	}()

	select {
	case <-done:
		assert.False(t, ok)
		return

	case <-time.After(100 * time.Millisecond):
		t.Error("Reconnect not triggered")
	}
}

func TestClose(t *testing.T) {
	c := testClient()
	close(c.quit)
	ok := false
	done := make(chan struct{})
	go func() {
		_, ok = <-c.Messages
		close(done)
	}()

	c.run()

	select {
	case <-done:
		assert.False(t, ok)
		return

	case <-time.After(100 * time.Millisecond):
		t.Error("Channels not closed")
	}
}

func waitConnAndClose(t *testing.T, c *Client) {
	done := make(chan struct{})
	quit := make(chan struct{})
	go func() {
		<-ircd.conn
		quit <- struct{}{}
		<-ircd.connClosed
		close(done)
	}()

	for {
		select {
		case <-done:
			return

		case <-quit:
			assert.True(t, c.Connected())
			c.Quit()

		case <-time.After(500 * time.Millisecond):
			t.Error("Took too long")
			return
		}
	}
}

var testCert = []byte(`-----BEGIN CERTIFICATE-----
MIIBdzCCASOgAwIBAgIBADALBgkqhkiG9w0BAQUwEjEQMA4GA1UEChMHQWNtZSBD
bzAeFw03MDAxMDEwMDAwMDBaFw00OTEyMzEyMzU5NTlaMBIxEDAOBgNVBAoTB0Fj
bWUgQ28wWjALBgkqhkiG9w0BAQEDSwAwSAJBAN55NcYKZeInyTuhcCwFMhDHCmwa
IUSdtXdcbItRB/yfXGBhiex00IaLXQnSU+QZPRZWYqeTEbFSgihqi1PUDy8CAwEA
AaNoMGYwDgYDVR0PAQH/BAQDAgCkMBMGA1UdJQQMMAoGCCsGAQUFBwMBMA8GA1Ud
EwEB/wQFMAMBAf8wLgYDVR0RBCcwJYILZXhhbXBsZS5jb22HBH8AAAGHEAAAAAAA
AAAAAAAAAAAAAAEwCwYJKoZIhvcNAQEFA0EAAoQn/ytgqpiLcZu9XKbCJsJcvkgk
Se6AbGXgSlq+ZCEVo0qIwSgeBqmsJxUu7NCSOwVJLYNEBO2DtIxoYVk+MA==
-----END CERTIFICATE-----`)

var testKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIBPAIBAAJBAN55NcYKZeInyTuhcCwFMhDHCmwaIUSdtXdcbItRB/yfXGBhiex0
0IaLXQnSU+QZPRZWYqeTEbFSgihqi1PUDy8CAwEAAQJBAQdUx66rfh8sYsgfdcvV
NoafYpnEcB5s4m/vSVe6SU7dCK6eYec9f9wpT353ljhDUHq3EbmE4foNzJngh35d
AekCIQDhRQG5Li0Wj8TM4obOnnXUXf1jRv0UkzE9AHWLG5q3AwIhAPzSjpYUDjVW
MCUXgckTpKCuGwbJk7424Nb8bLzf3kllAiA5mUBgjfr/WtFSJdWcPQ4Zt9KTMNKD
EUO0ukpTwEIl6wIhAMbGqZK3zAAFdq8DD2jPx+UJXnh0rnOkZBzDtJ6/iN69AiEA
1Aq8MJgTaYsDQWyU/hDq5YkDJc9e9DSCvUIzqxQWMQE=
-----END RSA PRIVATE KEY-----`)
