package main

import (
	"bytes"
	"fmt"
	"iconsole/services"
	"io"
	"os"
	"time"

	"github.com/urfave/cli"
)

func screenShotCommand(ctx *cli.Context) error {
	udid := ctx.String("UDID")

	device, err := getDevice(udid)
	if err != nil {
		return err
	}

	ss, err := services.NewScreenshotService(device)
	if err != nil {
		return err
	}
	defer ss.Close()

	buf := bytes.NewBuffer([]byte{})
	if err := ss.Shot(buf); err != nil {
		return err
	}

	fs, err := os.Create(fmt.Sprintf("Screenshot-%s.png", time.Now().Format("2006-01-02 15.04.05")))
	if err != nil {
		return err
	}
	defer fs.Close()
	io.Copy(fs, buf)
	return nil
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
