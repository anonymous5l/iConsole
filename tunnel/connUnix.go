// +build linux darwin freebsd openbsd android netbsd solaris

package tunnel

import "net"

func RawDial() (net.Conn, error) {
	return net.Dial("unix", "/var/run/usbmuxd")
}
