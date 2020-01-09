package main

import (
	"bytes"
	"fmt"
	"iconsole/tunnel"

	"github.com/urfave/cli"
)

func syslogAction(ctx *cli.Context) error {
	udid := ctx.String("UDID")

	return service("com.apple.syslog_relay", udid, func(conn *tunnel.MixConnection) error {
		buf := make([]byte, 0x19000)

		for {
			n, err := conn.Read(buf)
			if err != nil {
				return err
			}
			buf = bytes.Replace(buf[:n], []byte("\\^["), []byte("\x1B"), -1)
			fmt.Print(string(buf))
		}
	})
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
