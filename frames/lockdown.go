package frames

const (
	ProtocolVersion = "2"
)

type LockdownRequest struct {
	Label           string `plist:"Label"`
	ProtocolVersion string `plist:"ProtocolVersion"`
	Request         string `plist:"Request"`
}

type ValueRequest struct {
	LockdownRequest
	Domain string      `plist:"Domain,omitempty"`
	Key    string      `plist:"Key,omitempty"`
	Value  interface{} `plist:"Value,omitempty"`
}

type ValueResponse struct {
	LockdownRequest
	Domain string `plist:"Domain,omitempty"`
	Key    string `plist:"Key,omitempty"`
}

type LockdownResponse struct {
	Request string `plist:"Request"`
	Error   string `plist:"Error"`
}

type LockdownTypeResponse struct {
	LockdownResponse
	Type string `plist:"Type"`
}

type LockdownValueResponse struct {
	LockdownResponse
	Key   string      `plist:"Key"`
	Value interface{} `plist:"Value"`
}

func CreateLockdownRequest(request string) *LockdownRequest {
	return &LockdownRequest{
		Label:           BundleID,
		ProtocolVersion: ProtocolVersion,
		Request:         request,
	}
}

type StartSessionRequest struct {
	LockdownRequest
	SystemBUID string `plist:"SystemBUID"`
	HostID     string `plist:"HostID"`
}

type StopSessionRequest struct {
	LockdownRequest
	SessionID string `plist:"SessionID"`
}

type StartSessionResponse struct {
	LockdownResponse
	EnableSessionSSL bool   `plist:"EnableSessionSSL"`
	SessionID        string `plist:"SessionID"`
}

type PairRecordRequest struct {
	BaseRequest
	PairRecordID string `plist:"PairRecordID"`
}

type PairRecordResponse struct {
	Result
	PairRecordData []byte `plist:"PairRecordData"`
}

type PairRequest struct {
	LockdownRequest
	HostName       string                 `plist:"HostName"`
	PairRecord     *PairRecord            `plist:"PairRecord"`
	PairingOptions map[string]interface{} `plist:"PairingOptions"`
}

type PairRecord struct {
	DeviceCertificate []byte `plist:"DeviceCertificate"`
	EscrowBag         []byte `plist:"EscrowBag,omitempty"`
	HostCertificate   []byte `plist:"HostCertificate"`
	HostPrivateKey    []byte `plist:"HostPrivateKey"`
	HostID            string `plist:"HostID"`
	RootCertificate   []byte `plist:"RootCertificate"`
	RootPrivateKey    []byte `plist:"RootPrivateKey"`
	SystemBUID        string `plist:"SystemBUID"`
	WiFiMACAddress    string `plist:"WiFiMACAddress,omitempty"`
}

type StartServiceRequest struct {
	LockdownRequest
	Service   string `plist:"Service"`
	EscrowBag []byte `plist:"EscrowBag,omitempty"`
}

type StartServiceResponse struct {
	LockdownResponse
	EnableServiceSSL bool   `plist:"EnableServiceSSL"`
	Port             int    `plist:"Port"`
	Service          string `plist:"Service"`
}
