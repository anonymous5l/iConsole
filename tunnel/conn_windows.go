// +build windows

package tunnel

import "net"

func RawDial() (net.Conn, error) {
	return net.Dial("tcp", "localhost:27015")
}
