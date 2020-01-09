package main

import (
	"fmt"
	"iconsole/tunnel"
	"os"
	"time"

	"howett.net/plist"

	"github.com/urfave/cli"
)

func syncAndUnmarshal(service *tunnel.Service, obj interface{}) error {
	s, err := service.Sync()
	if err != nil {
		return err
	}
	if _, err := plist.Unmarshal(s.Body, obj); err != nil {
		return err
	}
	return nil
}

func screenShotCommand(ctx *cli.Context) error {
	udid := ctx.String("UDID")

	return service("com.apple.mobile.screenshotr", udid, func(conn *tunnel.MixConnection) error {

		service := tunnel.GenerateService(conn)

		firstMsg := []interface{}{
			"DLMessageVersionExchange",
			"DLVersionsOk",
		}

		captureMsg := []interface{}{
			"DLMessageProcessMessage",
			map[string]interface{}{
				"MessageType": "ScreenShotRequest",
			},
		}

		var f []interface{}

		// first accept version exchange
		if err := syncAndUnmarshal(service, &f); err != nil {
			return err
		}

		firstMsg = append(firstMsg, f[1])
		if err := service.SendBinary(firstMsg); err != nil {
			return err
		}

		if err := syncAndUnmarshal(service, &f); err != nil {
			return err
		}

		if f[3].(string) != "DLMessageDeviceReady" {
			return fmt.Errorf("message device not ready %s", f[3])
		}

		if err := service.SendBinary(captureMsg); err != nil {
			return err
		}

		if err := syncAndUnmarshal(service, &f); err != nil {
			return err
		}

		if f[4] != "DLMessageProcessMessage" {
			return fmt.Errorf("message device not ready %s %s", f[3], f[4])
		}

		screen := f[5].(map[string]interface{})
		if mt, ok := screen["MessageType"].(string); ok {
			if mt == "ScreenShotReply" {
				tiffData := screen["ScreenShotData"].([]byte)

				fs, err := os.Create(fmt.Sprintf("ScreenShot %s.tiff", time.Now().Format("2006-01-02 15.04.05")))

				if err != nil {
					return err
				}

				defer fs.Close()

				if _, err := fs.Write(tiffData); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("message type not `ScreenShotReply` %s", mt)
			}
		} else {
			return fmt.Errorf("message not found message type %#v", f)
		}

		return nil
	})
}

func initScreenShotCommond() cli.Command {
	return cli.Command{
		Name:      "screenshot",
		ShortName: "screen",
		Usage:     "Capture screen.",
		Action:    screenShotCommand,
		Flags:     globalFlags,
	}
}
