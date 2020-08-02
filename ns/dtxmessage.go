package ns

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

type DTXMessage struct {
	msgbuf *bytes.Buffer
}

func NewDTXMessage() *DTXMessage {
	return &DTXMessage{msgbuf: &bytes.Buffer{}}
}

func (this *DTXMessage) AppendObject(obj interface{}) error {
	archiver := NewNSKeyedArchiver()

	b, err := archiver.Marshal(obj)
	if err != nil {
		return err
	}

	this.AppendUInt32(10)
	this.AppendUInt32(2)
	this.AppendUInt32(uint32(len(b)))
	this.msgbuf.Write(b)

	return nil
}

func (this *DTXMessage) AppendInt64(v int64) {
	this.AppendUInt32(10)
	this.AppendUInt32(4)
	this.AppendUInt64(uint64(v))
}

func (this *DTXMessage) AppendInt32(v int32) {
	this.AppendUInt32(10)
	this.AppendUInt32(3)
	this.AppendUInt32(uint32(v))
}

func (this *DTXMessage) AppendUInt32(v uint32) {
	_ = binary.Write(this.msgbuf, binary.LittleEndian, v)
}

func (this *DTXMessage) AppendUInt64(v uint64) {
	_ = binary.Write(this.msgbuf, binary.LittleEndian, v)
}

func (this *DTXMessage) AppendBytes(b []byte) {
	this.msgbuf.Write(b)
}

func (this *DTXMessage) Len() int {
	return this.msgbuf.Len()
}

func (this *DTXMessage) ToBytes() []byte {
	dup := this.msgbuf.Bytes()
	b := make([]byte, 16)
	binary.LittleEndian.PutUint64(b, 0x1f0)
	binary.LittleEndian.PutUint64(b[8:], uint64(this.Len()))
	return append(b, dup...)
}

func UnmarshalDTXMessage(b []byte) ([]interface{}, error) {
	r := bytes.NewReader(b)
	var magic uint64
	var pkgLen uint64
	if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
		return nil, err
	} else if err := binary.Read(r, binary.LittleEndian, &pkgLen); err != nil {
		return nil, err
	}

	if magic != 0x1df0 {
		return nil, errors.New("magic not equal 0x1df0")
	}

	if pkgLen > uint64(len(b)-16) {
		return nil, errors.New("package length not enough")
	}

	var ret []interface{}

	for r.Len() > 0 {
		var flag uint32
		var typ uint32
		if err := binary.Read(r, binary.LittleEndian, &flag); err != nil {
			return nil, err
		} else if err := binary.Read(r, binary.LittleEndian, &typ); err != nil {
			return nil, err
		}
		switch typ {
		case 2:
			var l uint32
			if err := binary.Read(r, binary.LittleEndian, &l); err != nil {
				return nil, err
			}
			plistBuf := make([]byte, l)
			if _, err := r.Read(plistBuf); err != nil {
				return nil, err
			}
			archiver := NewNSKeyedArchiver()
			d, err := archiver.Unmarshal(plistBuf)
			if err != nil {
				return nil, err
			}
			ret = append(ret, d)
		case 3, 5:
			var i int32
			if err := binary.Read(r, binary.LittleEndian, &i); err != nil {
				return nil, err
			}
			ret = append(ret, i)
		case 4, 6:
			var i int64
			if err := binary.Read(r, binary.LittleEndian, &i); err != nil {
				return nil, err
			}
			ret = append(ret, i)
		case 10:
			// debug
			fmt.Println("Dictionary key!")
			continue
		default:
			// debug
			fmt.Printf("Unknow type %d\n", typ)
			break
		}
	}

	return ret, nil
}
