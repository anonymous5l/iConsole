package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"iconsole/frames"
	"iconsole/tunnel"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/urfave/cli"
)

type MountResponse struct {
	frames.LockdownResponse
	Status string `plist:"Status"`
}

type LookUpImageResponse struct {
	MountResponse
	ImageSignature [][]byte `plist:"ImageSignature"`
}

type MountRequest struct {
	Command   string `plist:"Command"`
	ImageType string `plist:"ImageType"`
}

type UploadImageRequest struct {
	MountRequest
	ImageSignature []byte `plist:"ImageSignature"`
	ImageSize      uint32 `plist:"ImageSize"`
}

type MountImageRequest struct {
	MountRequest
	ImagePath      string `plist:"ImagePath"`
	ImageSignature []byte `plist:"ImageSignature"`
}

func lookUpImages(service *tunnel.Service, imageType string) (*LookUpImageResponse, error) {

	var lookUpImageResponse LookUpImageResponse

	lookUpImageRequest := MountRequest{
		Command:   "LookupImage",
		ImageType: imageType,
	}

	if err := service.SendXML(lookUpImageRequest); err != nil {
		return nil, err
	}

	pkg, err := service.Sync()

	if err != nil {
		return nil, err
	}

	if err := pkg.UnmarshalBody(&lookUpImageResponse); err != nil {
		return nil, err
	}

	if lookUpImageResponse.Error != "" {
		return nil, errors.New(lookUpImageResponse.Error)
	}

	return &lookUpImageResponse, nil
}

func uploadImage(dmgFile, imageType string, signature []byte, baseConn *tunnel.MixConnection, service *tunnel.Service) error {
	dmgIo, err := os.Open(dmgFile)
	if err != nil {
		return err
	}
	defer dmgIo.Close()

	dmgIoLen := 0
	if ret, err := dmgIo.Seek(0, io.SeekEnd); err != nil {
		return err
	} else {
		dmgIoLen = int(ret)
	}

	if _, err := dmgIo.Seek(0, io.SeekStart); err != nil {
		return err
	}

	req := &UploadImageRequest{
		MountRequest: MountRequest{
			Command:   "ReceiveBytes",
			ImageType: imageType,
		},
		ImageSize:      uint32(dmgIoLen),
		ImageSignature: signature,
	}

	if err := service.SendXML(req); err != nil {
		return err
	}

	var resp MountResponse

	syncUnmarshalAndCheck := func() error {
		if pkg, err := service.Sync(); err != nil {
			return err
		} else if err := pkg.UnmarshalBody(&resp); err != nil {
			return err
		}

		if resp.Error != "" {
			return errors.New(resp.Error)
		}

		return nil
	}

	if err := syncUnmarshalAndCheck(); err != nil {
		return err
	}

	if resp.Status == "ReceiveBytesAck" {
		b := make([]byte, 0xffff)
		for {
			if n, err := dmgIo.Read(b); err != nil && err != io.EOF {
				return err
			} else if n > 0 {
				if _, err := baseConn.Write(b[:n]); err != nil {
					return err
				}
			} else {
				break
			}
		}
	}

	if err := syncUnmarshalAndCheck(); err != nil {
		return err
	}

	if resp.Status == "Complete" {
		return nil
	}

	return nil
}

func actionList(ctx *cli.Context) error {
	args := ctx.Args()
	udid := ctx.String("UDID")
	imageType := ctx.String("type")

	if len(args) > 0 {
		if strings.ToLower(args[0]) == "help" {
			return cli.ShowSubcommandHelp(ctx)
		}
	}

	return service("com.apple.mobile.mobile_image_mounter", udid, func(conn *tunnel.MixConnection) error {
		service := tunnel.GenerateService(conn)

		images, err := lookUpImages(service, imageType)
		if err != nil {
			return err
		}

		fmt.Printf("ImageSignatures[%d]:\n", len(images.ImageSignature))

		for i, is := range images.ImageSignature {
			fmt.Printf("%2d: %s\n", i, base64.StdEncoding.EncodeToString(is))
		}

		return nil
	})
}

func actionMount(ctx *cli.Context) error {
	udid := ctx.String("UDID")
	imageType := ctx.String("type")

	args := ctx.Args()

	var dmg_file, dmg_file_signature string

	if len(args) == 2 {
		dmg_file = args[0]
		dmg_file_signature = args[1]
	} else {
		return cli.ShowSubcommandHelp(ctx)
	}

	if _, err := os.Stat(dmg_file); err != nil {
		return err
	}
	if _, err := os.Stat(dmg_file_signature); err != nil {
		return err
	}

	var marjor int

	return session(udid, func(conn *tunnel.LockdownConnection) error {
		if len(conn.Version) >= 1 {
			marjor = conn.Version[0]
		}

		if marjor < 7 {
			return fmt.Errorf("Unsupported major version %d", marjor)
		}

		ssResp, err := conn.StartService("com.apple.mobile.mobile_image_mounter")
		if err != nil {
			return err
		}

		serviceConn, err := conn.GenerateConnection(ssResp.Port, ssResp.EnableServiceSSL)
		if err != nil {
			return err
		}
		defer serviceConn.Close()

		conn.Close()

		service := tunnel.GenerateService(serviceConn)

		dmgFileSignatureIo, err := os.Open(dmg_file_signature)
		if err != nil {
			return err
		}
		signature, err := ioutil.ReadAll(dmgFileSignatureIo)
		if err != nil {
			return err
		}
		if err := dmgFileSignatureIo.Close(); err != nil {
			return err
		}

		fmt.Println("Uploading image...")
		if err := uploadImage(dmg_file, imageType, signature, serviceConn, service); err != nil {
			return err
		}
		fmt.Println("Upload succeed...")

		req := MountImageRequest{
			MountRequest: MountRequest{
				Command:   "MountImage",
				ImageType: imageType,
			},
			ImagePath:      "/private/var/mobile/Media/PublicStaging/staging.dimage",
			ImageSignature: signature,
		}

		var resp MountResponse

		if err := service.SendXML(req); err != nil {
			return err
		}

		pkg, err := service.Sync()
		if err := pkg.UnmarshalBody(&resp); err != nil {
			return err
		}

		if resp.Error != "" {
			return errors.New((resp.Error))
		}

		if resp.Status != "Complete" {
			return fmt.Errorf("status: %s", resp.Status)
		}

		fmt.Println("Mount succeed")

		return nil
	})
}

func initMountCommand() cli.Command {
	flags := append(globalFlags, cli.StringFlag{
		Name:   "type, t",
		Usage:  "Image type default Developer",
		EnvVar: "IMAGE_TYPE",
		Value:  "Developer",
	})

	return cli.Command{
		Name:      "mount",
		Usage:     "Mount developer image",
		UsageText: "iconsole mount [-u serial_number|udid] <DMG_FILE> <DMG_FILE_SIGNATURE>",
		Action:    actionMount,
		Flags:     flags,
		Subcommands: []cli.Command{
			{
				Name:      "list",
				ShortName: "l",
				Usage:     "Show developer lists",
				Action:    actionList,
				Flags:     flags,
			},
		},
	}
}
