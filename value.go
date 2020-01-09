package main

import (
	"encoding/json"
	"fmt"
	"iconsole/tunnel"
	"strings"

	"github.com/urfave/cli"
)

func getAction(ctx *cli.Context) error {
	udid := ctx.String("UDID")

	args := ctx.Args()
	if len(args) == 0 {
		return nil
	}

	return session(udid, func(conn *tunnel.LockdownConnection) error {
		for _, s := range args {
			domain := ""
			key := ""
			a := strings.Split(s, ":")
			if len(a) == 2 {
				domain = a[0]
				key = a[1]
			} else if len(a) == 1 {
				key = a[0]
			} else {
				return fmt.Errorf("Arguments too many `%s`", s)
			}

			if resp, err := conn.GetValue(domain, key); err != nil {
				return err
			} else {
				if b, err := json.MarshalIndent(resp, "", "\t"); err != nil {
					return err
				} else {
					fmt.Println(string(b))
				}
			}
		}
		return nil
	})
}

func initValueCommond() cli.Command {
	return cli.Command{
		Name:   "get",
		Usage:  "Get session value",
		Action: getAction,
		Flags:  globalFlags,
	}
}
