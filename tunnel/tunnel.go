package tunnel

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"iconsole/frames"
	"net"

	"howett.net/plist"
)

var (
	ErrNoConnection = errors.New("not connection")
)

const (
	ResultOk          = 0
	ResultBadCommand  = 1
	ResultBadDev      = 2
	ResultCommRefused = 3
	// ???
	// ???
	ResultBadVersion = 6
	ResultUnknown    = 100
)

func getError(num uint64) error {
	switch num {
	case ResultOk:
		return nil
	case ResultBadCommand:
		return errors.New("BadCommand")
	case ResultBadDev:
		return errors.New("BadDev")
	case ResultCommRefused:
		return errors.New("CommRefused")
	case ResultBadVersion:
		return errors.New("BadVersion")
	default:
		return fmt.Errorf("ErrorCode %d", num)
	}
}

type PlistConnection struct {
	RawConn net.Conn
	version uint32
}

func NewPlistConnection() *PlistConnection {
	return &PlistConnection{
		version: 1,
	}
}

func (this *PlistConnection) Close() {
	if this.RawConn != nil {
		_ = this.RawConn.Close()
		this.RawConn = nil
	}
}

func (this *PlistConnection) Sync() (*frames.Package, error) {
	var err error
	var n int
	var pkg *frames.Package

	var pkgLen uint32
	if err = binary.Read(this.RawConn, binary.LittleEndian, &pkgLen); err != nil {
		return nil, err
	}

	pkgBuf := make([]byte, pkgLen)
	binary.LittleEndian.PutUint32(pkgBuf, pkgLen)

	offset := 4

	for {
		n, err = this.RawConn.Read(pkgBuf[offset:])
		if err != nil {
			return nil, err
		}
		if offset+n >= len(pkgBuf) {
			break
		}
		offset += n
	}

	pkg, err = frames.Unpack(pkgBuf)
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

func (this *PlistConnection) Dial() error {
	if conn, err := RawDial(); err != nil {
		return err
	} else {
		this.RawConn = conn
	}
	return nil
}

func (this *PlistConnection) Send(frame interface{}) error {
	if this.RawConn == nil {
		return ErrNoConnection
	}

	pkg := &frames.Package{
		Version: this.version,
		Type:    8,
		Tag:     0,
	}

	packageBuf, err := pkg.Pack(frame)
	if err != nil {
		return err
	}

	if _, err := this.RawConn.Write(packageBuf); err != nil {
		return err
	}

	return nil
}

func analyzeDevice(properties map[string]interface{}) (frames.Device, error) {
	ct := properties["ConnectionType"].(string)

	var device frames.Device

	model := frames.DeviceModel{
		ConnectionType: ct,
		DeviceID:       int(properties["DeviceID"].(uint64)),
		SerialNumber:   properties["SerialNumber"].(string),
	}

	switch ct {
	case "USB":
		device = &frames.USBDevice{
			DeviceModel:     model,
			ConnectionSpeed: int(properties["ConnectionSpeed"].(uint64)),
			LocationID:      int(properties["LocationID"].(uint64)),
			ProductID:       int(properties["ProductID"].(uint64)),
			UDID:            properties["UDID"].(string),
			USBSerialNumber: properties["USBSerialNumber"].(string),
		}
	case "Network":
		device = &frames.NetworkDevice{
			DeviceModel:            model,
			EscapedFullServiceName: properties["EscapedFullServiceName"].(string),
			InterfaceIndex:         int(properties["InterfaceIndex"].(uint64)),
			NetworkAddress:         properties["NetworkAddress"].([]uint8),
		}
	}

	return device, nil
}

// just for once call
func Devices() ([]frames.Device, error) {
	conn := NewPlistConnection()
	if err := conn.Dial(); err != nil {
		return nil, err
	}
	defer conn.Close()

	frame := frames.CreateBaseRequest(frames.ListDevices)

	if err := conn.Send(frame); err != nil {
		return nil, err
	}

	var devices []frames.Device
	var m map[string]interface{}

	respPkg, err := conn.Sync()
	if err != nil {
		return nil, err
	}
	if err := respPkg.UnmarshalBody(&m); err != nil {
		return nil, err
	}

	deviceList, ok := m["DeviceList"].([]interface{})
	if ok {
		for _, v := range deviceList {
			item := v.(map[string]interface{})
			properties := item["Properties"].(map[string]interface{})
			device, err := analyzeDevice(properties)
			if err != nil {
				return nil, err
			}
			devices = append(devices, device)
		}
	} else if n, ok := m["Number"].(uint64); ok {
		return nil, getError(n)
	} else {
		return nil, getError(ResultUnknown)
	}

	return devices, nil
}

func ReadBUID() (string, error) {
	conn := NewPlistConnection()
	if err := conn.Dial(); err != nil {
		return "", err
	}

	frame := frames.CreateBaseRequest("ReadBUID")

	if err := conn.Send(frame); err != nil {
		return "", err
	}

	defer conn.Close()

	pkg, err := conn.Sync()
	if err != nil {
		return "", err
	}

	var m map[string]interface{}
	if err := pkg.UnmarshalBody(&m); err != nil {
		return "", err
	}

	if buid, ok := m["BUID"].(string); ok {
		return buid, nil
	} else if n, ok := m["Number"].(uint64); ok {
		return "", getError(n)
	}

	return "", getError(ResultUnknown)
}

func Listen(msgNotifyer chan frames.Response) (context.CancelFunc, error) {
	conn := NewPlistConnection()
	if err := conn.Dial(); err != nil {
		return nil, err
	}

	frame := frames.CreateBaseRequest(frames.Listen)
	frame.LibUSBMuxVersion = frames.LibUSBMuxVersion

	if err := conn.Send(frame); err != nil {
		return nil, err
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	go func() {
		defer conn.Close()
		defer close(msgNotifyer)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				pkg, err := conn.Sync()
				if err != nil {
					return
				}

				var m map[string]interface{}
				if err := pkg.UnmarshalBody(&m); err != nil {
					return
				}

				if mt, ok := m["MessageType"].(string); ok {
					switch mt {
					case "Attached":
						device, err := analyzeDevice(m["Properties"].(map[string]interface{}))
						if err != nil {
							return
						}
						msgNotifyer <- &frames.DeviceAttached{
							BaseResponse: frames.BaseResponse{MessageType: mt},
							DeviceID:     int(m["DeviceID"].(uint64)),
							Properties:   device,
						}
					case "Detached":
						msgNotifyer <- &frames.DeviceDetached{
							BaseResponse: frames.BaseResponse{MessageType: mt},
							DeviceID:     int(m["DeviceID"].(uint64)),
						}
					case "Result":
						msgNotifyer <- &frames.Result{
							BaseResponse: frames.BaseResponse{MessageType: mt},
							Number:       int(m["Number"].(uint64)),
						}
					}
				}
			}
		}
	}()

	return cancelFunc, nil
}

func connectRaw(deviceId int, port int) (conn *PlistConnection, err error) {
	conn = NewPlistConnection()

	if err = conn.Dial(); err != nil {
		return
	}

	defer func() {
		if err != nil {
			conn.Close()
			conn = nil
		}
	}()

	connRequest := &frames.ConnectRequest{
		BaseRequest: *frames.CreateBaseRequest(frames.Connect),
		DeviceID:    deviceId,
		PortNumber:  ((port << 8) & 0xFF00) | (port >> 8),
	}

	if err = conn.Send(connRequest); err != nil {
		return
	}

	var pkg *frames.Package
	pkg, err = conn.Sync()
	if err != nil {
		return
	}

	var result frames.Result

	if err = pkg.UnmarshalBody(&result); err != nil {
		return
	}

	if result.Number == ResultOk {
		return
	}

	err = fmt.Errorf("errcode %d", result.Number)
	return
}

func Connect(device frames.Device, port int) (*PlistConnection, error) {
	return connectRaw(device.GetDeviceID(), port)
}

func readPairRecordRaw(udid string) (*frames.PairRecord, error) {
	conn := NewPlistConnection()
	if err := conn.Dial(); err != nil {
		return nil, err
	}

	frame := frames.CreateBaseRequest("ReadPairRecord")
	req := &frames.PairRecordRequest{
		BaseRequest:  *frame,
		PairRecordID: udid,
	}
	if err := conn.Send(req); err != nil {
		return nil, err
	}

	defer conn.Close()

	pkg, err := conn.Sync()
	if err != nil {
		return nil, err
	}

	var m frames.PairRecordResponse
	if err := pkg.UnmarshalBody(&m); err != nil {
		return nil, err
	}

	if m.Number != 0 {
		switch m.Number {
		case 1:
			return nil, errors.New("BadCommand")
		case 2:
			return nil, errors.New("BadDev")
		case 3:
			return nil, errors.New("ConnectionRefused")
		case 6:
			return nil, errors.New("BadVersion")
		}
	}

	var resp frames.PairRecord
	if _, err := plist.Unmarshal(m.PairRecordData, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func ReadPairRecord(device frames.Device) (*frames.PairRecord, error) {
	return readPairRecordRaw(device.GetSerialNumber())
}
