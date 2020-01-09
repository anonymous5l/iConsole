package frames

const (
	ProgramName      = "iConsole"
	ClientVersion    = "iConsole-Beta"
	BundleID         = "anonymous5l.iConsole"
	LibUSBMuxVersion = 3
)

const (
	Listen      = "Listen"
	Connect     = "Connect"
	ListDevices = "ListDevices"
)

type Device interface {
	GetConnectionType() string
	GetDeviceID() int
	GetSerialNumber() string
}

type Response interface {
	GetMessageType() string
}

type (
	BaseResponse struct {
		MessageType string `plist:"MessageType"`
	}

	BaseRequest struct {
		MessageType         string `plist:"MessageType"`
		BundleID            string `plist:"BundleID"`
		LibUSBMuxVersion    int    `plist:"kLibUSBMuxVersion,omitempty"`
		ClientVersionString string `plist:"ClientVersionString"`
		ProgramName         string `plist:"ProgName"`
	}

	ConnectRequest struct {
		BaseRequest
		DeviceID   int `plist:"DeviceID"`
		PortNumber int `plist:"PortNumber"`
	}

	DeviceModel struct {
		ConnectionType string `plist:"ConnectionType"`
		DeviceID       int    `plist:"DeviceID"`
		SerialNumber   string `plist:"SerialNumber"`
	}

	NetworkDevice struct {
		DeviceModel
		EscapedFullServiceName string `plist:"EscapedFullServiceName"`
		InterfaceIndex         int    `plist:"InterfaceIndex"`
		NetworkAddress         []byte `plist:"NetworkAddress"`
	}

	USBDevice struct {
		DeviceModel
		ConnectionSpeed int    `plist:"ConnectionSpeed"`
		LocationID      int    `plist:"LocationID"`
		ProductID       int    `plist:"ProductID"`
		UDID            string `plist:"UDID"`
		USBSerialNumber string `plist:"USBSerialNumber"`
	}

	DeviceAttached struct {
		BaseResponse
		DeviceID   int    `plist:"DeviceID"`
		Properties Device `plist:"Properties"`
	}

	DeviceDetached struct {
		BaseResponse
		DeviceID int `plist:"DeviceID"`
	}

	Result struct {
		BaseResponse
		Number int `plist:"Number"`
	}
)

func (this *DeviceModel) GetConnectionType() string {
	return this.ConnectionType
}

func (this *DeviceModel) GetDeviceID() int {
	return this.DeviceID
}

func (this *DeviceModel) GetSerialNumber() string {
	return this.SerialNumber
}

func (this *BaseResponse) GetMessageType() string {
	return this.MessageType
}

func CreateBaseRequest(mt string) *BaseRequest {
	return &BaseRequest{
		MessageType:         mt,
		BundleID:            BundleID,
		ClientVersionString: ClientVersion,
		ProgramName:         ProgramName,
	}
}
