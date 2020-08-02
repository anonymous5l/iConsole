package tunnel

import (
	"bytes"
	"encoding/binary"
	"iconsole/frames"
	"net"

	"howett.net/plist"
)

type Service struct {
	conn *MixConnection
}

func (this *Service) DismissSSL() error {
	return this.conn.DismissSSL()
}

func (this *Service) GetConnection() net.Conn {
	return this.conn
}

func (this *Service) Send(frame interface{}, format int) error {
	if this.conn == nil {
		return ErrNoConnection
	}

	pkg := &frames.ServicePackage{}

	packageBuf, err := pkg.Pack(frame, format)
	if err != nil {
		return err
	}

	_, err = this.conn.Write(packageBuf)
	return err
}

func (this *Service) SendXML(frame interface{}) error {
	return this.Send(frame, plist.XMLFormat)
}

func (this *Service) SendBinary(frame interface{}) error {
	return this.Send(frame, plist.BinaryFormat)
}

func (this *Service) Sync() (*frames.ServicePackage, error) {
	if this.conn == nil {
		return nil, ErrNoConnection
	}

	buf := bytes.NewBuffer([]byte{})

	var err error
	var n int
	var pkg *frames.ServicePackage

	offset := 0
	pkgLen := 0

	pkgBuf := make([]byte, 0x4)
	for {
		n, err = this.conn.Read(pkgBuf)

		if err != nil && n == 0 {
			return nil, err
		}

		buf.Write(pkgBuf[:n])

		offset += n

		if pkgLen == 0 {
			pkgLen = int(binary.BigEndian.Uint32(pkgBuf[:4])) + 4
			pkgBuf = make([]byte, pkgLen-4)
		}

		if offset >= pkgLen {
			break
		}
	}

	pkg, err = frames.UnpackLockdown(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return pkg, nil
}
