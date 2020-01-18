package services

import (
	"errors"
	"fmt"
	"iconsole/frames"
	"iconsole/tunnel"
)

type HouseArrestService struct {
	service *tunnel.Service
	afc     bool
}

func NewHouseArrestService(device frames.Device) (*HouseArrestService, error) {
	serv, err := startService(HouseArrestServiceName, device)
	if err != nil {
		return nil, err
	}

	return &HouseArrestService{service: serv}, nil
}

func (this *HouseArrestService) command(cmd, id string) error {
	m := map[string]string{
		"Command":    cmd,
		"Identifier": id,
	}

	if err := this.service.SendXML(m); err != nil {
		return err
	} else if pkg, err := this.service.Sync(); err != nil {
		return err
	} else {
		var resp map[string]interface{}
		if err := pkg.UnmarshalBody(&resp); err != nil {
			return err
		} else if e, ok := resp["Error"].(string); ok {
			return errors.New(e)
		} else if s, ok := resp["Status"].(string); !ok {
			return errors.New("unknown error")
		} else if s != "Complete" {
			return fmt.Errorf("status: %s", s)
		}
	}

	return nil
}

func (this *HouseArrestService) Documents(id string) (*AFCService, error) {
	if this.afc {
		return nil, errors.New("please use `AFCService`")
	}

	if err := this.command("VendDocuments", id); err != nil {
		return nil, err
	} else {
		this.afc = true
		return &AFCService{service: this.service}, nil
	}
}

func (this *HouseArrestService) Container(id string) (*AFCService, error) {
	if this.afc {
		return nil, errors.New("please use `AFCService`")
	}

	if err := this.command("VendContainer", id); err != nil {
		return nil, err
	} else {
		this.afc = true
		return &AFCService{service: this.service}, nil
	}
}
