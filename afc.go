package main

import (
	"bytes"
	"errors"
	"fmt"
	"iconsole/services"
	"io"
	"os"

	"github.com/urfave/cli"
)

func byteCountDecimal(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "kMGTPE"[exp])
}

func afcSpaceAction(ctx *cli.Context) error {
	udid := ctx.String("UDID")

	device, err := getDevice(udid)
	if err != nil {
		return err
	}

	afc, err := services.NewAFCService(device)
	if err != nil {
		return err
	}
	defer afc.Close()

	info, err := afc.GetDeviceInfo()
	if err != nil {
		return err
	}

	fmt.Printf("      Model: %s\n", info.Model)
	fmt.Printf("  BlockSize: %d\n", info.BlockSize/8)
	fmt.Printf("  FreeSpace: %s\n", byteCountDecimal(int64(info.FreeBytes)))
	fmt.Printf("  UsedSpace: %s\n", byteCountDecimal(int64(info.TotalBytes-info.FreeBytes)))
	fmt.Printf(" TotalSpace: %s\n", byteCountDecimal(int64(info.TotalBytes)))

	return nil
}

func afcLsAction(ctx *cli.Context) error {
	udid := ctx.String("UDID")

	args := ctx.Args()
	if len(args) <= 0 {
		return cli.ShowSubcommandHelp(ctx)
	}

	device, err := getDevice(udid)
	if err != nil {
		return err
	}

	afc, err := services.NewAFCService(device)
	if err != nil {
		return err
	}
	defer afc.Close()

	if p, err := afc.ReadDirectory(args[0], true); err != nil {
		return err
	} else {
		for _, v := range p {
			if i, err := afc.GetFileInfo(v); err != nil {
				return err
			} else if i.IsDir() {
				fmt.Printf("%7s %s \x1B[1;34m%s\x1B[0m\n", byteCountDecimal(i.Size()), i.ModTime().Format("2006-01-02 15:04:05"), i.Name())
			} else {
				fmt.Printf("%7s %s %s\n", byteCountDecimal(i.Size()), i.ModTime().Format("2006-01-02 15:04:05"), i.Name())
			}
		}
	}

	return nil
}

func printTree(afc *services.AFCService, raw string, hasNexts []bool) error {
	if p, err := afc.ReadDirectory(raw, true); err != nil {
		return err
	} else {
		for i, v := range p {
			b := &bytes.Buffer{}
			for _, hasNext := range hasNexts {
				if hasNext {
					b.WriteString("│   ")
				} else {
					b.WriteString("    ")
				}
			}
			var name string
			info, err := afc.GetFileInfo(v)
			if err != nil {
				return err
			} else if info.IsDir() {
				name = fmt.Sprintf("\x1B[1;34m%s\x1B[0m", info.Name())
				//printTree(afc, v)
			} else {
				name = fmt.Sprintf("%s", info.Name())
			}

			lastIndex := len(p) - 1

			// print tree
			if i == lastIndex {
				b.WriteString("└──")
				b.WriteString(name)
			} else {
				b.WriteString("├──")
				b.WriteString(name)
			}

			fmt.Println(string(b.Bytes()))

			if info.IsDir() {
				if i == lastIndex {
					hasNexts = append(hasNexts, false)
				} else {
					hasNexts = append(hasNexts, true)
				}
				err := printTree(afc, v, hasNexts)
				hasNexts = hasNexts[:len(hasNexts)-1]
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func afcTreeAction(ctx *cli.Context) error {
	udid := ctx.String("UDID")

	args := ctx.Args()
	if len(args) <= 0 {
		return cli.ShowSubcommandHelp(ctx)
	}

	device, err := getDevice(udid)
	if err != nil {
		return err
	}

	afc, err := services.NewAFCService(device)
	if err != nil {
		return err
	}
	defer afc.Close()

	return printTree(afc, args[0], []bool{})
}

func afcUploadAction(ctx *cli.Context) error {
	udid := ctx.String("UDID")

	args := ctx.Args()
	if len(args) < 2 {
		return cli.ShowSubcommandHelp(ctx)
	}

	src := args[0]
	dst := args[1]

	if fi, err := os.Stat(src); err != nil {
		return err
	} else if fi.IsDir() {
		return errors.New("for now only support file not directory")
	}

	device, err := getDevice(udid)
	if err != nil {
		return err
	}

	afc, err := services.NewAFCService(device)
	if err != nil {
		return err
	}
	defer afc.Close()

	f, err := afc.FileOpen(dst, services.AFC_RW)
	if err != nil {
		return err
	}
	defer f.Close()

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if _, err := io.Copy(f, srcFile); err != nil {
		return err
	}

	return nil
}

func afcDownloadAction(ctx *cli.Context) error {
	udid := ctx.String("UDID")

	args := ctx.Args()
	if len(args) < 2 {
		return cli.ShowSubcommandHelp(ctx)
	}

	src := args[1]
	dst := args[0]

	device, err := getDevice(udid)
	if err != nil {
		return err
	}

	afc, err := services.NewAFCService(device)
	if err != nil {
		return err
	}
	defer afc.Close()

	fi, err := afc.GetFileInfo(dst)
	if err != nil {
		return err
	} else if fi.IsDir() {
		return errors.New("for now only support file not directory")
	}

	f, err := afc.FileOpen(dst, services.AFC_RDONLY)
	if err != nil {
		return err
	}
	defer f.Close()

	srcFile, err := os.Create(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if _, err := io.Copy(srcFile, f); err != nil {
		return err
	}

	return nil
}

func initAFCCommand() cli.Command {
	return cli.Command{
		Name:  "afc",
		Usage: "Apple file conduit",
		Flags: globalFlags,
		Subcommands: []cli.Command{
			{
				Name:      "space",
				ShortName: "s",
				Usage:     "Device space usage detail",
				Action:    afcSpaceAction,
				Flags:     globalFlags,
			},
			{
				Name:   "dir",
				Usage:  "dir <path>",
				Action: afcLsAction,
				Flags:  globalFlags,
			},
			{
				Name:   "tree",
				Usage:  "tree <path>",
				Action: afcTreeAction,
				Flags:  globalFlags,
			},
			{
				Name:   "upload",
				Usage:  "Upload <src file path> <dst file path>",
				Action: afcUploadAction,
				Flags:  globalFlags,
			},
			{
				Name:   "download",
				Usage:  "download <dst file path> <src file path>",
				Action: afcDownloadAction,
				Flags:  globalFlags,
			},
		},
	}
}
