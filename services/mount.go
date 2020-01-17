package services

import (
	"errors"
	"fmt"
	"iconsole/frames"
	"iconsole/tunnel"
	"io"
	"io/ioutil"
	"os"
)

type MountResponse struct {
	frames.LockdownResponse
	Status string `plist:"Status"`
}

type Images struct {
	MountResponse
	ImageSignature [][]byte `plist:"ImageSignature"`
}

type MountRequest struct {
	Command   string `plist:"Command"`
	ImageType string `plist:"ImageType"`
}

type uploadImageRequest struct {
	MountRequest
	ImageSignature []byte `plist:"ImageSignature"`
	ImageSize      uint32 `plist:"ImageSize"`
}

type mountImageRequest struct {
	MountRequest
	ImagePath      string `plist:"ImagePath"`
	ImageSignature []byte `plist:"ImageSignature"`
}

type MountService struct {
	service *tunnel.Service
}

func NewMountService(device frames.Device) (*MountService, error) {
	serv, err := startService(MountServiceName, device)
	if err != nil {
		return nil, err
	}

	return &MountService{service: serv}, nil
}

func (this *MountService) Images(imageType string) (*Images, error) {
	req := MountRequest{
		Command:   "LookupImage",
		ImageType: imageType,
	}

	if err := this.service.SendXML(req); err != nil {
		return nil, err
	}

	var resp Images

	if err := syncServiceAndCheckError(this.service, &resp); err != nil {
		return nil, err
	} else if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}

	return &resp, nil
}

func (this *MountService) readFileData(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

func (this *MountService) UploadImage(dmg, signature, imageType string) error {
	dmgFile, err := os.Open(dmg)
	if err != nil {
		return err
	}
	defer func() {
		if dmgFile != nil {
			dmgFile.Close()
		}
	}()

	signatureData, err := this.readFileData(signature)
	if err != nil {
		return err
	}

	dmgFileSize, err := dmgFile.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	if _, err := dmgFile.Seek(0, io.SeekStart); err != nil {
		return err
	}

	req := &uploadImageRequest{
		MountRequest: MountRequest{
			Command:   "ReceiveBytes",
			ImageType: imageType,
		},
		ImageSize:      uint32(dmgFileSize),
		ImageSignature: signatureData,
	}

	if err := this.service.SendXML(req); err != nil {
		return err
	}

	var resp MountResponse
	if err := syncServiceAndCheckError(this.service, &resp); err != nil {
		return err
	} else if resp.Error != "" {
		return errors.New(resp.Error)
	}

	if resp.Status != "ReceiveBytesAck" {
		return fmt.Errorf("status: %s", resp.Status)
	}

	b := make([]byte, 0xffff)
	baseConn := this.service.GetConnection()
	for {
		if n, err := dmgFile.Read(b); err != nil && err != io.EOF {
			return err
		} else if n > 0 {
			if _, err := baseConn.Write(b[:n]); err != nil {
				return err
			}
		} else {
			break
		}
	}
	dmgFile.Close()
	dmgFile = nil

	if err := syncServiceAndCheckError(this.service, &resp); err != nil {
		return err
	} else if resp.Error != "" {
		return errors.New(resp.Error)
	}

	if resp.Status != "Complete" {
		return fmt.Errorf("status: %s", resp.Status)
	}

	return nil
}

func (this *MountService) Mount(path, imageType, signature string) error {
	signatureData, err := this.readFileData(signature)
	if err != nil {
		return err
	}

	req := mountImageRequest{
		MountRequest: MountRequest{
			Command:   "MountImage",
			ImageType: imageType,
		},
		ImagePath:      path,
		ImageSignature: signatureData,
	}

	var resp MountResponse

	if err := this.service.SendXML(req); err != nil {
		return err
	} else if err := syncServiceAndCheckError(this.service, &resp); err != nil {
		return err
	} else if resp.Error != "" {
		return errors.New(resp.Error)
	} else if resp.Status != "Complete" {
		return fmt.Errorf("status: %s", resp.Status)
	}

	return nil
}

func (this *MountService) Close() error {
	return this.service.GetConnection().Close()
}
