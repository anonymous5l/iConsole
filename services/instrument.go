package services

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"iconsole/frames"
	"iconsole/ns"
	"iconsole/tunnel"
	"time"
	"unsafe"
)

type DTXMessagePayloadHeader struct {
	Flags           uint32
	AuxiliaryLength uint32
	TotalLength     uint64
}

func (this DTXMessagePayloadHeader) Marshal() []byte {
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, this.Flags)
	_ = binary.Write(buf, binary.LittleEndian, this.AuxiliaryLength)
	_ = binary.Write(buf, binary.LittleEndian, this.TotalLength)
	return buf.Bytes()
}

func (this *DTXMessagePayloadHeader) Unmarshal(buf []byte) error {
	r := bytes.NewReader(buf)
	if err := binary.Read(r, binary.LittleEndian, &this.Flags); err != nil {
		return err
	} else if err := binary.Read(r, binary.LittleEndian, &this.AuxiliaryLength); err != nil {
		return err
	} else if err := binary.Read(r, binary.LittleEndian, &this.TotalLength); err != nil {
		return err
	}
	return nil
}

type PrivateResponseMessage struct {
	Aux []interface{}
	Obj interface{}
}

type DTXMessageHeader struct {
	Magic             uint32
	CB                uint32
	FragmentId        uint16
	FragmentCount     uint16
	Length            uint32
	Identifier        uint32
	ConversationIndex uint32
	ChannelCode       uint32
	ExpectsReply      uint32
}

func (this *DTXMessageHeader) Unmarshal(data []byte) error {
	r := bytes.NewReader(data)
	if err := binary.Read(r, binary.LittleEndian, &this.Magic); err != nil {
		return err
	} else if err := binary.Read(r, binary.LittleEndian, &this.CB); err != nil {
		return err
	} else if err := binary.Read(r, binary.LittleEndian, &this.FragmentId); err != nil {
		return err
	} else if err := binary.Read(r, binary.LittleEndian, &this.FragmentCount); err != nil {
		return err
	} else if err := binary.Read(r, binary.LittleEndian, &this.Length); err != nil {
		return err
	} else if err := binary.Read(r, binary.LittleEndian, &this.Identifier); err != nil {
		return err
	} else if err := binary.Read(r, binary.LittleEndian, &this.ConversationIndex); err != nil {
		return err
	} else if err := binary.Read(r, binary.LittleEndian, &this.ChannelCode); err != nil {
		return err
	} else if err := binary.Read(r, binary.LittleEndian, &this.ExpectsReply); err != nil {
		return err
	}
	return nil
}

func (this DTXMessageHeader) Marshal() []byte {
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, this.Magic)
	_ = binary.Write(buf, binary.LittleEndian, this.CB)
	_ = binary.Write(buf, binary.LittleEndian, this.FragmentId)
	_ = binary.Write(buf, binary.LittleEndian, this.FragmentCount)
	_ = binary.Write(buf, binary.LittleEndian, this.Length)
	_ = binary.Write(buf, binary.LittleEndian, this.Identifier)
	_ = binary.Write(buf, binary.LittleEndian, this.ConversationIndex)
	_ = binary.Write(buf, binary.LittleEndian, this.ChannelCode)
	_ = binary.Write(buf, binary.LittleEndian, this.ExpectsReply)
	return buf.Bytes()
}

type InstrumentService struct {
	service     *tunnel.Service
	hs          bool
	msgId       uint32
	channels    map[string]int32
	openChannel map[string]uint32
}

func NewInstrumentService(device frames.Device) (*InstrumentService, error) {
	service, err := startService(InstrumentsServiceName, device)
	if err != nil {
		return nil, err
	}

	if err := service.DismissSSL(); err != nil {
		return nil, err
	}

	return &InstrumentService{
		service:     service,
		channels:    make(map[string]int32),
		openChannel: make(map[string]uint32),
	}, nil
}

func (this *InstrumentService) recvPrivateMessage() (*PrivateResponseMessage, error) {
	payloadBuf := &bytes.Buffer{}
	for {
		header := &DTXMessageHeader{}
		headerBuf := make([]byte, unsafe.Sizeof(*header))
		if _, err := this.service.GetConnection().Read(headerBuf); err != nil {
			return nil, err
		}

		if err := header.Unmarshal(headerBuf); err != nil {
			return nil, err
		}

		if header.Magic != 0x1F3D5B79 {
			return nil, fmt.Errorf("bad magic %x", header.Magic)
		}

		if header.ConversationIndex == 1 {
			if header.Identifier != this.msgId {
				return nil, fmt.Errorf("except identifier %d new identifier %d", this.msgId, header.Identifier)
			}
		} else if header.ConversationIndex == 0 {
			if header.Identifier > this.msgId {
				this.msgId = header.Identifier
			} else if header.Identifier < this.msgId {
				return nil, fmt.Errorf("unexcept identifier %d", header.Identifier)
			}
		} else {
			return nil, fmt.Errorf("invalid conversationIndex %d", header.ConversationIndex)
		}

		if header.FragmentId == 0 {
			if header.FragmentCount > 1 {
				continue
			}
		}

		nRecv := 0
		for nRecv < int(header.Length) {
			_cap := 2048
			left := int(header.Length) - nRecv
			if left < _cap {
				_cap = left
			}
			recvBuf := make([]byte, _cap)
			n, err := this.service.GetConnection().Read(recvBuf)
			if err != nil {
				return nil, err
			}
			payloadBuf.Write(recvBuf[:n])
			nRecv += n
		}

		if header.FragmentId == header.FragmentCount-1 {
			break
		}
	}

	payloadBytes := payloadBuf.Bytes()
	payload := &DTXMessagePayloadHeader{}
	if err := payload.Unmarshal(payloadBytes); err != nil {
		return nil, err
	}

	compress := (payload.Flags & 0xff000) >> 12
	if compress != 0 {
		return nil, fmt.Errorf("message is compressed type %d", compress)
	}

	payloadSize := uint32(unsafe.Sizeof(*payload))
	objOffset := payloadSize + payload.AuxiliaryLength

	aux := payloadBytes[payloadSize : payloadSize+payload.AuxiliaryLength]
	obj := payloadBytes[objOffset : uint64(objOffset)+(payload.TotalLength-uint64(payload.AuxiliaryLength))]

	ret := &PrivateResponseMessage{}

	if len(aux) > 0 {
		if aux, err := ns.UnmarshalDTXMessage(aux); err != nil {
			return nil, err
		} else {
			ret.Aux = aux
		}
	}

	if len(obj) > 0 {
		if obj, err := ns.NewNSKeyedArchiver().Unmarshal(obj); err != nil {
			return nil, err
		} else {
			ret.Obj = obj
		}
	}

	return ret, nil
}

func (this *InstrumentService) sendPrivateMessage(selector string, args *ns.DTXMessage, channel uint32, expectReply bool) error {
	payload := &DTXMessagePayloadHeader{}
	header := &DTXMessageHeader{
		ExpectsReply: 1,
	}

	er := 0x1000
	if !expectReply {
		er = 0
		header.ExpectsReply = 0
	}

	sel, err := ns.NewNSKeyedArchiver().Marshal(selector)
	if err != nil {
		return err
	}

	aux := make([]byte, 0)

	if args != nil {
		aux = args.ToBytes()
	}

	payload.Flags = uint32(0x2 | er)
	payload.AuxiliaryLength = uint32(len(aux))
	payload.TotalLength = uint64(len(aux)) + uint64(len(sel))

	header.Magic = 0x1F3D5B79
	header.CB = uint32(unsafe.Sizeof(*header))
	header.FragmentId = 0
	header.FragmentCount = 1
	header.Length = uint32(unsafe.Sizeof(*payload)) + uint32(payload.TotalLength)
	this.msgId++
	header.Identifier = this.msgId
	header.ConversationIndex = 0
	header.ChannelCode = channel

	msgBuf := &bytes.Buffer{}
	msgBuf.Write(header.Marshal())
	msgBuf.Write(payload.Marshal())
	msgBuf.Write(aux)
	msgBuf.Write(sel)

	_, err = this.service.GetConnection().Write(msgBuf.Bytes())
	return err
}

func (this *InstrumentService) makeChannel(channel string) (uint32, error) {
	if _, ok := this.channels[channel]; !ok {
		return 0, fmt.Errorf("not support %s", channel)
	} else if c, ok := this.openChannel[channel]; ok {
		return c, nil
	} else {
		c := uint32(len(this.openChannel) + 1)
		msg := ns.NewDTXMessage()
		msg.AppendInt32(int32(c))
		if err := msg.AppendObject(channel); err != nil {
			return 0, err
		}

		if err := this.sendPrivateMessage("_requestChannelWithCode:identifier:", msg, 0, true); err != nil {
			return 0, err
		}

		if _, err := this.recvPrivateMessage(); err != nil {
			return 0, err
		}

		return c, nil
	}
}

type Application struct {
	AppExtensionUUIDs         []string
	BundlePath                string
	CFBundleIdentifier        string
	ContainerBundleIdentifier string
	ContainerBundlePath       string
	PluginIdentifier          string
	PluginUUID                string
	DisplayName               string
	ExecutableName            string
	Placeholder               string
	Restricted                int
	Type                      string
	Version                   string
}

func (this *InstrumentService) AppList() ([]Application, error) {
	c, err := this.makeChannel("com.apple.instruments.server.services.device.applictionListing")
	if err != nil {
		return nil, err
	}

	// could use filter
	emptyMap := make(map[string]interface{})
	msg := ns.NewDTXMessage()
	if err := msg.AppendObject(emptyMap); err != nil {
		return nil, err
	}
	if err := msg.AppendObject(""); err != nil {
		return nil, err
	}

	if err := this.sendPrivateMessage("installedApplicationsMatching:registerUpdateToken:", msg, c, true); err != nil {
		return nil, err
	}

	resp, err := this.recvPrivateMessage()
	if err != nil {
		return nil, err
	}

	var apps []Application
	rapps := resp.Obj.([]interface{})
	for _, v := range rapps {
		m := v.(map[string]interface{})
		a := &Application{}
		if uuids, ok := m["AppExtensionUUIDs"].([]interface{}); ok {
			for _, uuid := range uuids {
				a.AppExtensionUUIDs = append(a.AppExtensionUUIDs, uuid.(string))
			}
		}
		a.BundlePath, _ = m["BundlePath"].(string)
		a.CFBundleIdentifier, _ = m["CFBundleIdentifier"].(string)
		a.DisplayName, _ = m["DisplayName"].(string)
		a.ExecutableName, _ = m["ExecutableName"].(string)
		a.Placeholder, _ = m["Placeholder"].(string)
		a.ContainerBundleIdentifier, _ = m["ContainerBundleIdentifier"].(string)
		a.ContainerBundlePath, _ = m["ContainerBundlePath"].(string)
		a.PluginIdentifier, _ = m["PluginIdentifier"].(string)
		a.PluginUUID, _ = m["PluginUUID"].(string)
		a.Restricted = int(m["Restricted"].(uint64))
		a.Type, _ = m["Type"].(string)
		a.Version, _ = m["Version"].(string)
		apps = append(apps, *a)
	}

	return apps, nil
}

type Process struct {
	IsApplication bool
	Name          string
	Pid           int
	RealAppName   string
	StartDate     time.Time
}

func (this *InstrumentService) ProcessList() ([]Process, error) {
	c, err := this.makeChannel("com.apple.instruments.server.services.deviceinfo")
	if err != nil {
		return nil, err
	}

	if err := this.sendPrivateMessage("runningProcesses", nil, c, true); err != nil {
		return nil, err
	}

	resp, err := this.recvPrivateMessage()
	if err != nil {
		return nil, err
	}

	var p []Process

	objs := resp.Obj.([]interface{})
	for _, v := range objs {
		m := v.(map[string]interface{})
		tp := &Process{}
		if m["isApplication"].(bool) {
			tp.IsApplication = true
		}
		tp.Name = m["name"].(string)
		tp.Pid = int(m["pid"].(uint64))
		tp.RealAppName = m["realAppName"].(string)
		if t, ok := m["startDate"].(time.Time); ok {
			tp.StartDate = t
		}

		p = append(p, *tp)
	}

	return p, nil
}

func (this *InstrumentService) Kill(pid int) error {
	c, err := this.makeChannel("com.apple.instruments.server.services.processcontrol")
	if err != nil {
		return err
	}

	msg := ns.NewDTXMessage()
	if err := msg.AppendObject(pid); err != nil {
		return err
	}

	if err := this.sendPrivateMessage("killPid:", msg, c, false); err != nil {
		return err
	}

	return nil
}

func (this *InstrumentService) Launch(bundleId string) (int, error) {
	c, err := this.makeChannel("com.apple.instruments.server.services.processcontrol")
	if err != nil {
		return 0, err
	}

	msg := ns.NewDTXMessage()
	// application path: not use through empty
	if err := msg.AppendObject(""); err != nil {
		return 0, err
	}
	// `CFBundleIdentifier`
	if err := msg.AppendObject(bundleId); err != nil {
		return 0, err
	}
	// launch app environment variables: not use
	if err := msg.AppendObject(map[string]interface{}{}); err != nil {
		return 0, err
	}
	// launch app start arguments: not use
	if err := msg.AppendObject([]interface{}{}); err != nil {
		return 0, err
	}
	// launch app options
	if err := msg.AppendObject(map[string]interface{}{
		"StartSuspendedKey": 0,
		"KillExisting":      1,
	}); err != nil {
		return 0, err
	}

	if err := this.sendPrivateMessage("launchSuspendedProcessWithDevicePath:bundleIdentifier:environment:arguments:options:", msg, c, true); err != nil {
		return 0, err
	}

	resp, err := this.recvPrivateMessage()
	if err != nil {
		return 0, err
	}

	if err, ok := resp.Obj.(ns.GoNSError); ok {
		return 0, fmt.Errorf("%s", err.NSUserInfo.(map[string]interface{})["NSLocalizedDescription"])
	}

	return int(resp.Obj.(uint64)), nil
}

func (this *InstrumentService) Handshake() error {
	if !this.hs {
		msg := ns.NewDTXMessage()
		if err := msg.AppendObject(map[string]interface{}{
			"com.apple.private.DTXBlockCompression": 2,
			"com.apple.private.DTXConnection":       1,
		}); err != nil {
			return err
		}

		if err := this.sendPrivateMessage("_notifyOfPublishedCapabilities:", msg, 0, false); err != nil {
			return err
		}

		resp, err := this.recvPrivateMessage()
		if err != nil {
			return err
		}

		if resp.Obj.(string) != "_notifyOfPublishedCapabilities:" {
			return fmt.Errorf("response obj %s", resp.Obj)
		}

		aux := resp.Aux[0].(map[string]interface{})
		for k, v := range aux {
			this.channels[k] = int32(v.(uint64))
		}
	}

	return nil
}
