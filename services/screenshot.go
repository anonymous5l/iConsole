package services

import (
	"errors"
	"fmt"
	"iconsole/frames"
	"iconsole/tunnel"
	"io"
)

type ScreenshotService struct {
	service *tunnel.Service
	hs      bool
}

func NewScreenshotService(device frames.Device) (*ScreenshotService, error) {
	serv, err := startService(ScreenshotServiceName, device)
	if err != nil {
		return nil, err
	}

	return &ScreenshotService{service: serv}, nil
}

func (this *ScreenshotService) Handshake() error {
	if !this.hs {
		firstMsg := []interface{}{
			"DLMessageVersionExchange",
			"DLVersionsOk",
		}

		var f []interface{}
		if err := syncServiceAndCheckError(this.service, &f); err != nil {
			return err
		}
		firstMsg = append(firstMsg, f[1])
		if err := this.service.SendBinary(firstMsg); err != nil {
			return err
		} else if err := syncServiceAndCheckError(this.service, &f); err != nil {
			return err
		} else if f[3].(string) != "DLMessageDeviceReady" {
			return fmt.Errorf("message device not ready %s", f[3])
		}
		this.hs = true
	}
	return nil
}

func (this *ScreenshotService) Shot(w io.Writer) error {
	if err := this.Handshake(); err != nil {
		return err
	}

	var f []interface{}

	captureMsg := []interface{}{
		"DLMessageProcessMessage",
		map[string]interface{}{
			"MessageType": "ScreenShotRequest",
		},
	}

	if err := this.service.SendBinary(captureMsg); err != nil {
		return err
	} else if err := syncServiceAndCheckError(this.service, &f); err != nil {
		return err
	} else if f[0] != "DLMessageProcessMessage" {
		return fmt.Errorf("message device not ready %s %s", f[3], f[4])
	}

	screen := f[1].(map[string]interface{})
	if data, ok := screen["ScreenShotData"].([]byte); !ok {
		return errors.New("`ScreenShotData` not ready")
	} else if _, err := w.Write(data); err != nil {
		return err
	}

	return nil
}

func (this *ScreenshotService) Close() error {
	return this.service.GetConnection().Close()
}
