package main

import (
	"errors"
	"fmt"
	"iconsole/tunnel"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/urfave/cli"
)

func transportAction(ctx *cli.Context) error {
	udid := ctx.String("UDID")
	port := ctx.Int64("port")
	args := ctx.Args()
	t := "tcp"
	a := ""
	if len(args) == 2 {
		t = strings.ToLower(args[0])
		a = args[1]
	} else {
		return fmt.Errorf("Arguments not enough\nuseage example.\n\ticonsole relay -u <xxx> -p 1234 tcp :1234\n\ticonsole relay -u <xxx> -p 1234 tcp 127.0.0.1:443\n\ticonsole relay -u <xxx> -p 1234 unix /opt/debugserver")
	}

	l, err := net.Listen(t, a)
	if err != nil {
		return err
	}

	if t == "unix" {
		if err := os.Chmod(a, 0777); err != nil {
			return err
		}

		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)
		go func(c chan os.Signal) {
			<-c
			l.Close()
		}(sigc)
	}

	if udid == "" {
		return errors.New("Exec failed unset `UDID` argument")
	}

	if port < 3 {
		return errors.New("Port number range error")
	}

	device, err := getDevice(udid)
	if err != nil {
		return err
	}

	for {
		come, err := l.Accept()
		if err != nil {
			return nil
		}

		go func(front net.Conn) {
			back, err := tunnel.Connect(device, int(port))
			if err != nil {
				front.Close()
				fmt.Printf("Dial device port `%d` error: %s\n", port, err)
				return
			}

			/* clean timeout */
			if err := back.RawConn.SetDeadline(time.Time{}); err != nil {
				fmt.Printf("SetDeadline %s\n", err)
				return
			}

			fmt.Printf("Accept new connection %s\n", front.RemoteAddr())
			go func(front, back net.Conn) {
				if _, err := io.Copy(front, back); err != nil {
					_ = front.Close()
					_ = back.Close()
				}
			}(front, back.RawConn)
			go func(front, back net.Conn) {
				if _, err := io.Copy(back, front); err != nil {
					_ = front.Close()
					_ = back.Close()
				}
			}(front, back.RawConn)
		}(come)
	}
}

func initTransportCommand() cli.Command {
	return cli.Command{
		Name:   "relay",
		Usage:  "Transport communication wrap to local network",
		Action: transportAction,
		Flags: append(globalFlags, cli.Int64Flag{
			Name:   "port, p",
			Usage:  "Connect to device port",
			EnvVar: "PORT",
		}),
	}
}
