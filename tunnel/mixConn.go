package tunnel

import (
	"crypto/tls"
	"errors"
	"iconsole/frames"
	"net"
	"time"
)

type MixConnection struct {
	conn net.Conn
	ssl  *tls.Conn
}

func MixConnectionClient(conn net.Conn) *MixConnection {
	return &MixConnection{
		conn: conn,
	}
}

func (this *MixConnection) DismissSSL() {
	this.ssl = nil
}

func (this *MixConnection) Handshake(version []int, record *frames.PairRecord) error {
	if record == nil {
		return errors.New("record nil")
	}

	minVersion := uint16(tls.VersionTLS11)
	maxVersion := uint16(tls.VersionTLS11)

	if version[0] > 10 {
		minVersion = tls.VersionTLS11
		maxVersion = tls.VersionTLS13
	}

	cert, err := tls.X509KeyPair(record.RootCertificate, record.RootPrivateKey)
	if err != nil {
		return err
	}

	cfg := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
		MinVersion:         minVersion,
		MaxVersion:         maxVersion,
	}

	this.ssl = tls.Client(this.conn, cfg)

	if err := this.ssl.Handshake(); err != nil {
		return err
	}

	return nil
}

func (this *MixConnection) getConn() net.Conn {
	if this.ssl != nil {
		return this.ssl
	}
	return this.conn
}

func (this *MixConnection) Read(b []byte) (n int, err error) {
	return this.getConn().Read(b)
}

func (this *MixConnection) Write(b []byte) (n int, err error) {
	return this.getConn().Write(b)
}

func (this *MixConnection) Close() error {
	this.ssl = nil
	return this.getConn().Close()
}

func (this *MixConnection) LocalAddr() net.Addr {
	return this.getConn().LocalAddr()
}

func (this *MixConnection) RemoteAddr() net.Addr {
	return this.getConn().RemoteAddr()
}

func (this *MixConnection) SetDeadline(t time.Time) error {
	return this.getConn().SetDeadline(t)
}

func (this *MixConnection) SetReadDeadline(t time.Time) error {
	return this.getConn().SetReadDeadline(t)
}

func (this *MixConnection) SetWriteDeadline(t time.Time) error {
	return this.getConn().SetWriteDeadline(t)
}
