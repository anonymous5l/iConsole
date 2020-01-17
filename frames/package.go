package frames

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"howett.net/plist"
)

type ServicePackage struct {
	Length uint32
	Body   []byte
}

func (this *ServicePackage) Pack(body interface{}, format int) ([]byte, error) {
	frameXml, err := plist.MarshalIndent(body, format, "\t")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	l := make([]byte, 4)
	binary.BigEndian.PutUint32(l, uint32(len(frameXml)))
	buf.Write(l)
	buf.Write(frameXml)

	return buf.Bytes(), nil
}

func (this *ServicePackage) UnmarshalBody(pkg interface{}) error {
	_, err := plist.Unmarshal(this.Body, pkg)
	return err
}

func (this *ServicePackage) String() string {
	return string(this.Body)
}

func UnpackLockdown(rawBytes []byte) (*ServicePackage, error) {
	pkg := &ServicePackage{}
	pkg.Length = binary.BigEndian.Uint32(rawBytes[:4])
	if len(rawBytes[4:]) != int(pkg.Length) {
		return nil, errors.New("buffer not enough")
	}
	pkg.Body = rawBytes[4:]
	return pkg, nil
}

type Package struct {
	Length  uint32
	Version uint32
	Type    uint32
	Tag     uint32
	Body    []byte
}

func (this *Package) Pack(body interface{}) ([]byte, error) {
	frameXml, err := plist.MarshalIndent(body, plist.XMLFormat, "\t")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	l := make([]byte, 4)
	binary.LittleEndian.PutUint32(l, uint32(len(frameXml)+16))
	buf.Write(l)
	binary.LittleEndian.PutUint32(l, this.Version)
	buf.Write(l)
	binary.LittleEndian.PutUint32(l, this.Type) // xml plist
	buf.Write(l)
	binary.LittleEndian.PutUint32(l, this.Tag) // pkg len
	buf.Write(l)
	buf.Write(frameXml)

	return buf.Bytes(), nil
}

func (this *Package) String() string {
	return fmt.Sprintf("Length: %d Version: %d Type: %d Tag: %d\nBody: %s",
		this.Length, this.Version, this.Type, this.Tag,
		this.Body)
}

func (this *Package) UnmarshalBody(pkg interface{}) error {
	_, err := plist.Unmarshal(this.Body, pkg)
	return err
}

func Unpack(rawBytes []byte) (*Package, error) {
	pkg := &Package{}
	pkg.Length = binary.LittleEndian.Uint32(rawBytes[:4])
	if len(rawBytes) != int(pkg.Length) {
		return nil, errors.New("buffer not enough")
	}
	pkg.Version = binary.LittleEndian.Uint32(rawBytes[4:8])
	pkg.Type = binary.LittleEndian.Uint32(rawBytes[8:12])
	pkg.Tag = binary.LittleEndian.Uint32(rawBytes[12:16])
	pkg.Body = rawBytes[16:]
	return pkg, nil
}
