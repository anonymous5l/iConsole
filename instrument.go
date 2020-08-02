package main

import (
	"errors"
	"fmt"
	"iconsole/services"
	"os"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"

	"github.com/urfave/cli"
)

func actionProcessList(ctx *cli.Context) error {
	udid := ctx.String("UDID")

	device, err := getDevice(udid)
	if err != nil {
		return err
	}

	s, err := services.NewInstrumentService(device)
	if err != nil {
		return err
	}

	if err := s.Handshake(); err != nil {
		return err
	}

	p, err := s.ProcessList()
	if err != nil {
		return err
	}

	writer := tablewriter.NewWriter(os.Stdout)
	writer.SetHeader([]string{"PID", "ProcessName", "StartDate"})
	for _, v := range p {
		writer.Append([]string{
			strconv.Itoa(v.Pid),
			v.Name,
			v.StartDate.Format("2006-01-02 15:04:05"),
		})
	}
	writer.Render()

	return nil
}

func actionAppList(ctx *cli.Context) error {
	udid := ctx.String("UDID")

	device, err := getDevice(udid)
	if err != nil {
		return err
	}

	s, err := services.NewInstrumentService(device)
	if err != nil {
		return err
	}

	if err := s.Handshake(); err != nil {
		return err
	}

	p, err := s.AppList()
	if err != nil {
		return err
	}

	writer := tablewriter.NewWriter(os.Stdout)
	writer.SetHeader([]string{"Name", "BundleID"})
	for _, a := range p {
		writer.Append([]string{
			a.DisplayName,
			a.CFBundleIdentifier,
		})
	}
	writer.Render()

	return nil
}

func actionKill(ctx *cli.Context) error {
	udid := ctx.String("UDID")
	pid := ctx.Int("pid")

	if pid <= 0 {
		return errors.New("argument error")
	}

	device, err := getDevice(udid)
	if err != nil {
		return err
	}

	s, err := services.NewInstrumentService(device)
	if err != nil {
		return err
	}

	if err := s.Handshake(); err != nil {
		return err
	}

	if err := s.Kill(pid); err != nil {
		return err
	}

	return nil
}

func actionLaunch(ctx *cli.Context) error {
	udid := ctx.String("UDID")
	bundleId := ctx.String("bundleid")

	if strings.Trim(bundleId, "") == "" {
		return errors.New("argument error")
	}

	device, err := getDevice(udid)
	if err != nil {
		return err
	}

	s, err := services.NewInstrumentService(device)
	if err != nil {
		return err
	}

	if err := s.Handshake(); err != nil {
		return err
	}

	pid, err := s.Launch(bundleId)
	if err != nil {
		return err
	}

	fmt.Printf("Launch successful pid: %d\n", pid)

	return nil
}

func initProcessCommond() cli.Command {
	return cli.Command{
		Name:      "instrument",
		Usage:     "Instrument tools",
		UsageText: "iconsole instrument [-u serial_number|UDID] <Command>",
		Flags:     globalFlags,
		Subcommands: []cli.Command{
			{
				Name:      "proclist",
				ShortName: "pl",
				Usage:     "List all process",
				Action:    actionProcessList,
				Flags:     globalFlags,
			},
			{
				Name:      "applist",
				ShortName: "al",
				Usage:     "List all installed app",
				Action:    actionAppList,
				Flags:     globalFlags,
			},
			{
				Name:      "kill",
				ShortName: "k",
				Usage:     "Kill application",
				Action:    actionKill,
				Flags: append(globalFlags, cli.IntFlag{
					Name:     "pid, p",
					Usage:    "Process id",
					EnvVar:   "PROCESS_PID",
					Required: true,
				}),
			},
			{
				Name:      "launch",
				ShortName: "s",
				Usage:     "Launch application",
				Action:    actionLaunch,
				Flags: append(globalFlags, cli.StringFlag{
					Name:     "bundleid, i",
					Usage:    "Application bundle id",
					EnvVar:   "BUNDLE_ID",
					Required: true,
				}),
			},
		},
	}
}
