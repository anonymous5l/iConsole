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

		if !cb(this, buf[:n]) {
			break
		}
	}

	return this.Close()
}

func (this *SyslogRelayService) Close() error {
	this.closed = true
	return this.service.GetConnection().Close()
}
