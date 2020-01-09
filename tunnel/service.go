package tunnel

import (
	"encoding/binary"
	"iconsole/frames"

	"howett.net/plist"
)

type Service struct {
	conn *MixConnection
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

	if _, err := this.conn.Write(packageBuf); err != nil {
		return err
	}

	return nil
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

	var err error
	var n int
	var pkg *frames.ServicePackage

	var pkgLen uint32
	if err = binary.Read(this.conn, binary.BigEndian, &pkgLen); err != nil {
		return nil, err
	}

	pkgBuf := make([]byte, pkgLen+4)
	binary.BigEndian.PutUint32(pkgBuf, pkgLen)

	offset := 4

	for {
		n, err = this.conn.Read(pkgBuf[offset:])
		if err != nil {
			return nil, err
		}
		if offset+n >= len(pkgBuf) {
			break
		}
		offset += n
	}

	pkg, err = frames.UnpackLockdown(pkgBuf)
	if err != nil {
		return nil, err
	}

	return pkg, nil
}
