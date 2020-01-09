package main

import (
	"fmt"
	"iconsole/tunnel"

	"github.com/urfave/cli"
)

func devicesAction(ctx *cli.Context) error {
	devices, err := tunnel.Devices()
	if err != nil {
		return err
	}
	for _, d := range devices {
		conn, err := tunnel.LockdownDial(d)
		if err != nil {
			return err
		}

		if err := conn.StartSession(); err != nil {
			return err
		}

		deviceName, err := conn.DeviceName()
		if err != nil {
			return err
		}
		deviceType, err := conn.DeviceClass()
		if err != nil {
			return err
		}
		version, err := conn.ProductVersion()
		if err != nil {
			return err
		}

		fmt.Printf("%s %s %s\n\tConnectionType: %s\n\tUDID: %s\n", deviceType, deviceName, version, d.GetConnectionType(), d.GetSerialNumber())

		if err := conn.StopSession(); err != nil {
			return err
		}

		conn.Close()
	}
	return nil
}

func initDevices() cli.Command {
	return cli.Command{
		Name:      "devices",
		ShortName: "dev",
		Usage:     "List all connect devices",
		Action:    devicesAction,
	}
}
