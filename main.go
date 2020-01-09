package main

import (
	"errors"
	"fmt"
	"iconsole/frames"
	"iconsole/tunnel"
	"os"

	"github.com/urfave/cli"
)

var globalFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "UDID, u",
		Usage:  "device serialNumber UDID",
		EnvVar: "DEVICE_UDID",
		Value:  "",
	},
}

func session(udid string, cb func(*tunnel.LockdownConnection) error) error {
	if udid == "" {
		return errors.New("exec failed unset `UDID` argument")
	}

	device, err := getDevice(udid)
	if err != nil {
		return err
	}

	conn, err := tunnel.LockdownDial(device)
	if err != nil {
		return err
	}

	defer conn.Close()

	if err := conn.StartSession(); err != nil {
		return err
	}

	defer conn.StopSession()

	return cb(conn)
}

func service(service string, udid string, cb func(*tunnel.MixConnection) error) error {
	return session(udid, func(conn *tunnel.LockdownConnection) error {
		resp, err := conn.StartService(service)
		if err != nil {
			return err
		}

		serviceConn, err := conn.GenerateConnection(resp.Port, resp.EnableServiceSSL)
		if err != nil {
			return err
		}

		defer serviceConn.Close()

		if err := cb(serviceConn); err != nil {
			return err
		}

		return nil
	})
}

func getDevice(udid string) (frames.Device, error) {
	devices, err := tunnel.Devices()
	if err != nil {
		return nil, err
	}

	var device frames.Device

	for i, d := range devices {
		if d.GetSerialNumber() == udid {
			device = devices[i]
			break
		}
	}

	if device == nil {
		return nil, fmt.Errorf("device %s was not found", udid)
	}

	return device, nil
}

func main() {
	app := cli.NewApp()
	app.Name = "iConsole"
	app.Usage = "iOS device tools"
	app.Version = "1.0.0-Alpha"
	app.Authors = []cli.Author{
		{
			Name:  "anonymous5l",
			Email: "wxdxfg@hotmail.com",
		},
	}
	app.Commands = []cli.Command{
		initDevices(),
		initSyslogCommond(),
		initSimCommond(),
		initScreenShotCommond(),
		initSyncCommond(),
		initValueCommond(),
		initTransportCommand(),
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err.Error())
		return
	}
}
