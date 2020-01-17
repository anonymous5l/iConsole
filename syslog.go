package main

import (
	"bytes"
	"fmt"
	"iconsole/services"

	"github.com/urfave/cli"
)

func logCb(_ *services.SyslogRelayService, log []byte) bool {
	repl := string(bytes.Replace(log, []byte{0x5c, 0x5e, 0x5b}, []byte{0x1b}, -1))
	fmt.Print(repl)
	return true
}

func syslogAction(ctx *cli.Context) error {
	udid := ctx.String("UDID")

	if device, err := getDevice(udid); err != nil {
		return err
	} else if srs, err := services.NewSyslogRelayService(device); err != nil {
		return err
	} else if err := srs.Relay(logCb); err != nil {
		return err
	}

	return nil
}

func initSyslogCommond() cli.Command {
	return cli.Command{
		Name:      "syslog",
		ShortName: "log",
		Usage:     "Relay syslog of a connected device.",
		Action:    syslogAction,
		Flags:     globalFlags,
	}
}
