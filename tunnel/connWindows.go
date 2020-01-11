// +build windows

package tunnel

import (
	"net"
	"time"
)

func RawDial(timeout time.Duration) (net.Conn, error) {
	dialer := net.Dialer{
		Timeout: timeout,
	}

	return dialer.Dial("tcp", "127.0.0.1:27015")
}
