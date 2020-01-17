package main

import (
	"encoding/base64"
	"fmt"
	"iconsole/services"
	"strings"

	"github.com/urfave/cli"
)

func actionList(ctx *cli.Context) error {
	args := ctx.Args()
	udid := ctx.String("UDID")
	imageType := ctx.String("type")

	if len(args) > 0 {
		if strings.ToLower(args[0]) == "help" {
			return cli.ShowSubcommandHelp(ctx)
		}
	}

	if device, err := getDevice(udid); err != nil {
		return err
	} else if ms, err := services.NewMountService(device); err != nil {
		return err
	} else if images, err := ms.Images(imageType); err != nil {
		return err
	} else {
		fmt.Printf("ImageSignatures[%d]:\n", len(images.ImageSignature))

		for i, is := range images.ImageSignature {
			fmt.Printf("%2d: %s\n", i, base64.StdEncoding.EncodeToString(is))
		}
	}

	return nil
}

func actionMount(ctx *cli.Context) error {
	udid := ctx.String("UDID")
	imageType := ctx.String("type")

	args := ctx.Args()

	var dmgFile, dmgFileSignature string

	if len(args) == 2 {
		dmgFile = args[0]
		dmgFileSignature = args[1]
	} else {
		return cli.ShowSubcommandHelp(ctx)
	}

	path := "/private/var/mobile/Media/PublicStaging/staging.dimage"

	if device, err := getDevice(udid); err != nil {
		return err
	} else if ms, err := services.NewMountService(device); err != nil {
		return err
	} else if err := ms.UploadImage(dmgFile, dmgFileSignature, imageType); err != nil {
		return err
	} else if err := ms.Mount(path, imageType, dmgFileSignature); err != nil {
		return err
	}

	return nil
}

func initMountCommand() cli.Command {
	flags := append(globalFlags, cli.StringFlag{
		Name:   "type, t",
		Usage:  "Image type default Developer",
		EnvVar: "IMAGE_TYPE",
		Value:  "Developer",
	})

	return cli.Command{
		Name:      "mount",
		Usage:     "Mount developer image",
		UsageText: "iconsole mount [-u serial_number|udid] <DMG_FILE> <DMG_FILE_SIGNATURE>",
		Action:    actionMount,
		Flags:     flags,
		Subcommands: []cli.Command{
			{
				Name:      "list",
				ShortName: "l",
				Usage:     "Show developer lists",
				Action:    actionList,
				Flags:     flags,
			},
		},
	}
}
