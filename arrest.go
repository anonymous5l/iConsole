package main

import (
	"fmt"
	"iconsole/services"
	"path"

	"github.com/urfave/cli"
)

func arrestAction(ctx *cli.Context) error {
	udid := ctx.String("UDID")

	device, err := getDevice(udid)
	if err != nil {
		return err
	}

	args := ctx.Args()
	if len(args) <= 0 {
		return cli.ShowSubcommandHelp(ctx)
	}

	a, err := services.NewHouseArrestService(device)
	if err != nil {
		return err
	}
	afc, err := a.Documents(args[0])
	if err != nil {
		return err
	}

	base := "Documents"

	if p, err := afc.ReadDirectory(base); err != nil {
		return err
	} else {
		for _, v := range p {
			if v != "." && v != ".." {
				if i, err := afc.GetFileInfo(path.Join(base, v)); err != nil {
					return err
				} else if i.IsDir() {
					fmt.Printf("%7s %s \x1B[1;34m%s\x1B[0m\n", byteCountDecimal(i.Size()), i.ModTime().Format("2006-01-02 15:04:05"), i.Name())
				} else {
					fmt.Printf("%7s %s %s\n", byteCountDecimal(i.Size()), i.ModTime().Format("2006-01-02 15:04:05"), i.Name())
				}
			}
		}
	}

	return nil
}

func initArrest() cli.Command {
	return cli.Command{
		Name:      "arrest",
		Usage:     "House arrest",
		UsageText: "iconsole arrest <BundleID>",
		Action:    arrestAction,
	}
}
