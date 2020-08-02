package main

import (
	"fmt"
	"iconsole/services"
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

	fs, err := os.Create(fmt.Sprintf("Screenshot-%s.png", time.Now().Format("2006-01-02 15.04.05")))
	if err != nil {
		return err
	}
	defer fs.Close()
	if err := ss.Shot(fs); err != nil {
		return err
	}

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
