package main

import (
	"fmt"
	"iconsole/tunnel"

	"github.com/urfave/cli"
)

func edAction(ctx *cli.Context, ed bool) error {
	udid := ctx.String("UDID")

	return session(udid, func(conn *tunnel.LockdownConnection) error {
		if resp, err := conn.SetValue("com.apple.mobile.wireless_lockdown", "EnableWifiConnections", ed); err != nil {
			return err
		} else if resp.Value.(bool) == ed {
			fmt.Println("Succeed")
		} else {
			fmt.Println("Failed")
		}

		return nil
	})
}

func syncEnableAction(ctx *cli.Context) error {
	return edAction(ctx, true)
}

func syncDisableAction(ctx *cli.Context) error {
	return edAction(ctx, false)
}

func syncAction(ctx *cli.Context) error {
	udid := ctx.String("UDID")

	return session(udid, func(conn *tunnel.LockdownConnection) error {
		if resp, err := conn.GetValue("com.apple.mobile.wireless_lockdown", "EnableWifiConnections"); err != nil {
			return err
		} else if resp.Value.(bool) {
			fmt.Println("Device enable WiFi connections")
		} else {
			fmt.Println("Device disable WiFi connections")
		}

		return nil
	})
}

func initSyncCommond() cli.Command {
	return cli.Command{
		Name:   "sync",
		Usage:  "Enable Wi-Fi sync or disable",
		Action: syncAction,
		Flags:  globalFlags,
		Subcommands: []cli.Command{
			{
				Name:      "enable",
				ShortName: "e",
				Action:    syncEnableAction,
				Flags:     globalFlags,
			},
			{
				Name:      "disable",
				ShortName: "d",
				Action:    syncDisableAction,
				Flags:     globalFlags,
			},
		},
	}
}
