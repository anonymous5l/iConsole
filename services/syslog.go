package services

import (
	"errors"
	"iconsole/frames"
	"iconsole/tunnel"
)

type SyslogRelayService struct {
	service *tunnel.Service
	closed  bool
}

func NewSyslogRelayService(device frames.Device) (*SyslogRelayService, error) {
	serv, err := startService(SyslogRelayServiceName, device)
	if err != nil {
		return nil, err
	}

	return &SyslogRelayService{service: serv}, nil
}

func (this *SyslogRelayService) IsClosed() bool {
	return this.closed
}

const (
	kBackslash = 0x5c
	kM         = 0x4d
	kDash      = 0x2d
	kCaret     = 0x5e
	kNum       = 0x30
)

func (this *SyslogRelayService) isDigit(d []byte) bool {
	for i := 0; i < len(d); i++ {
		if (d[i] & 0xf0) != kNum {
			return false
		}
	}
	return true
}

func (this *SyslogRelayService) unicode(data []byte) []byte {
	var out []byte
	for i := 0; i < len(data); {
		if data[i] != kBackslash || i > len(data)-4 {
			out = append(out, data[i])
			i++
		} else {
			if data[i+1] == kM && data[i+2] == kCaret {
				out = append(out, (data[i+3]&0x7f)+0x40)
			} else if data[i+1] == kCaret {
				// don't know is right
				out = append(out, (data[i+2]&0x7f)-0x40, data[i+3])
			} else if data[i+1] == kM && data[i+2] == kDash {
				out = append(out, data[i+3]|0x80)
			} else if this.isDigit(data[i+1 : i+3]) {
				out = append(out, (data[i+1]&0x3)<<6|(data[i+2]&0x7)<<3|data[i+3]&0x07)
			} else {
				out = append(out, data[i:i+4]...)
			}
			i += 4
		}
	}
	return out
}

func (this *SyslogRelayService) Relay(cb func(*SyslogRelayService, []byte) bool) error {
	if this.IsClosed() {
		return errors.New("closed")
	}

	buf := make([]byte, 0xffff)

	for {
		n, err := this.service.GetConnection().Read(buf)
		if err != nil && n == 0 {
			return err
		}

		if !cb(this, this.unicode(buf[:n])) {
			break
		}
	}

	return this.Close()
}

func (this *SyslogRelayService) Close() error {
	this.closed = true
	return this.service.GetConnection().Close()
}
