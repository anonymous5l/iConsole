// +build linux darwin freebsd openbsd android netbsd solaris

package tunnel

import (
	"net"
	"time"
)

func RawDial(timeout time.Duration) (net.Conn, error) {
	dialer := net.Dialer{
		Timeout: timeout,
	}

	return dialer.Dial("unix", "/var/run/usbmuxd")
}
