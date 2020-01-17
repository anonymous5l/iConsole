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

	pkgBuf := make([]byte, 0xffff)

	var err error
	var n int
	var pkg *frames.ServicePackage

	offset := 0
	pkgLen := 0

	for {
		n, err = this.conn.Read(pkgBuf[offset:])
		if err != nil && n == 0 {
			return nil, err
		}
		offset += n
		if pkgLen == 0 {
			pkgLen = int(binary.BigEndian.Uint32(pkgBuf[:4])) + 4
		}
		if offset >= pkgLen {
			break
		}
	}

	pkg, err = frames.UnpackLockdown(pkgBuf[:offset])
	if err != nil {
		return nil, err
	}

	return pkg, nil
}
