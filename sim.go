package main

import (
	"iconsole/services"

	"github.com/urfave/cli"
)

func startSimAction(ctx *cli.Context) error {
	udid := ctx.String("UDID")
	lon := ctx.Float64("lon")
	lat := ctx.Float64("lat")
	coor := ctx.String("coor")

	device, err := getDevice(udid)
	if err != nil {
		return err
	} else if sls, err := services.NewSimulateLocationService(device); err != nil {
		return err
	} else if err := sls.Start(lon, lat, coor); err != nil {
		return err
	}

	return err
}

func stopSimAction(ctx *cli.Context) error {
	udid := ctx.String("UDID")

	device, err := getDevice(udid)
	if err != nil {
		return err
	} else if sls, err := services.NewSimulateLocationService(device); err != nil {
		return err
	} else if err := sls.Stop(); err != nil {
		return err
	}

	return err
}

func initSimCommond() cli.Command {
	return cli.Command{
		Name:      "simlocation",
		ShortName: "sim",
		Usage:     "A mounted developer disk image is required on the device.",
		Subcommands: []cli.Command{
			{
				Name:   "start",
				Action: startSimAction,
				Flags: append(globalFlags, cli.Float64Flag{
					Name:     "latitude, lat",
					EnvVar:   "LATITUDE",
					Required: true,
					Value:    0,
				},
					cli.Float64Flag{
						Name:     "longtitude, lon",
						EnvVar:   "LONGTITUDE",
						Required: true,
						Value:    0,
					},
					cli.StringFlag{
						Name:   "coordinate, coor",
						EnvVar: "COORDINATE",
						Usage:  "coordinate name `gcj02` `wsg84` `bd09` default `gcj02`",
						Value:  "gcj02",
					}),
			},
			{
				Name:   "stop",
				Action: stopSimAction,
				Flags:  globalFlags,
			},
		},
	}
}
