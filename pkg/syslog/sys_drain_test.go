package syslog_test

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"

	"code.cloudfoundry.org/rfc5424"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type spySink struct {
	lis net.Listener
}

func newTLSSpySink(addr ...string) *spySink {
	a := ":0"
	if len(addr) != 0 {
		a = addr[0]
	}

	cert, err := tls.X509KeyPair(tlsCert, tlsKey)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	lis, err := tls.Listen("tcp", a, config)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return &spySink{
		lis: lis,
	}
}

func newSpySink(addr ...string) *spySink {
	a := ":0"
	if len(addr) != 0 {
		a = addr[0]
	}
	lis, err := net.Listen("tcp", a)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return &spySink{
		lis: lis,
	}
}

func (s *spySink) url() string {
	return s.lis.Addr().String()
}

func (s *spySink) stop() {
	_ = s.lis.Close()
}

func (s *spySink) accept() net.Conn {
	conn, err := s.lis.Accept()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return conn
}

func (s *spySink) expectReceived(msgs ...string) {
	conn := s.accept()
	defer func() {
		_ = conn.Close()
	}()
	buf := bufio.NewReader(conn)

	for _, expected := range msgs {
		expected = fmt.Sprintf("%d %s", len(expected), expected)
		actual, err := buf.ReadString('\n')
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		ExpectWithOffset(1, actual).To(Equal(expected))
	}
}

func (s *spySink) expectReceivedOnly(msgs ...string) {
	conn := s.accept()
	defer func() {
		_ = conn.Close()
	}()
	buf := bufio.NewReader(conn)

	for _, expected := range msgs {
		expected = fmt.Sprintf("%d %s", len(expected), expected)
		actual, err := buf.ReadString('\n')
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		ExpectWithOffset(1, actual).To(Equal(expected))
	}

	read := make(chan struct{})
	go func() {
		defer close(read)
		_, _ = buf.ReadString('\n')
	}()
	select {
	case <-read:
		Fail("unexpected read occurred", 1)
	case <-time.After(300 * time.Millisecond):
	}
}

func (s *spySink) expectReceivedWithSD(sds ...[]rfc5424.StructuredData) {
	conn := s.accept()
	defer func() {
		_ = conn.Close()
	}()
	buf := bufio.NewReader(conn)

	for _, expectedSD := range sds {
		lenB, err := buf.ReadBytes(' ')
		Expect(err).ToNot(HaveOccurred())
		length, err := strconv.Atoi(string(lenB[:len(lenB)-1]))
		Expect(err).ToNot(HaveOccurred())

		data := make([]byte, length)
		n, err := buf.Read(data)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(length))

		var msg rfc5424.Message
		err = msg.UnmarshalBinary(data)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		ExpectWithOffset(1, msg.StructuredData).To(Equal(expectedSD))
	}
}
