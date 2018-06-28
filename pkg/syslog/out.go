package syslog

import (
	"bytes"
	"net"
	"time"

	"code.cloudfoundry.org/rfc5424"
)

type Out struct {
	addr string
	conn net.Conn
}

func NewOut(addr string) *Out {
	return &Out{
		addr: addr,
	}
}

func (o *Out) Write(
	record map[string]string,
	ts time.Time,
	tag string,
) error {
	err := o.maintainConnection()
	if err != nil {
		return err
	}

	msg := convert(record, ts, tag)
	_, err = msg.WriteTo(o.conn)
	if err != nil {
		return err
	}
	return nil
}

func (o *Out) maintainConnection() error {
	if o.conn == nil {
		conn, err := net.Dial("tcp", o.addr)
		o.conn = conn
		return err
	}
	return nil
}

func convert(
	record map[string]string,
	ts time.Time,
	tag string,
) *rfc5424.Message {
	return &rfc5424.Message{
		Priority:  rfc5424.Info + rfc5424.User,
		Timestamp: ts,
		Message:   appendNewline([]byte(record["log"])),
	}
}

func appendNewline(msg []byte) []byte {
	if !bytes.HasSuffix(msg, []byte("\n")) {
		msg = append(msg, byte('\n'))
	}
	return msg
}
