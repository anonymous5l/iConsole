package tunnel

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"iconsole/frames"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
)

const (
	LockdownPort = 62078
)

type LockdownConnection struct {
	conn       *Service
	version    []int
	device     frames.Device
	pairRecord *frames.PairRecord
	sslSession *frames.StartSessionResponse
}

func getPemCertificate(cert []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := pem.Encode(buf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func getPairPemFormat(cert []byte, key *rsa.PrivateKey) ([]byte, []byte, error) {
	p, err := getPemCertificate(cert)
	if err != nil {
		return nil, nil, err
	}

	buf := new(bytes.Buffer)
	if err := pem.Encode(buf, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}); err != nil {
		return nil, nil, err
	}

	priv := buf.Bytes()

	return p, priv, nil
}

func (this *LockdownConnection) generatePairRecord() (*frames.PairRecord, error) {
	record := &frames.PairRecord{}

	buid, err := ReadBUID()
	if err != nil {
		return nil, err
	}

	valueResp, err := this.GetValue("", "DevicePublicKey")
	if err != nil {
		return nil, err
	}

	if valueResp.Value == nil {
		return nil, fmt.Errorf("%s", valueResp.Error)
	}

	block, _ := pem.Decode(valueResp.Value.([]byte))
	deviceKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	hostKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("Failed to generate serial number: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * (24 * 365) * 10)

	rootTemplate := x509.Certificate{
		IsCA:                  true,
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    x509.SHA1WithRSA,
		PublicKeyAlgorithm:    x509.RSA,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		BasicConstraintsValid: true,
	}

	caCert, err := x509.CreateCertificate(rand.Reader, &rootTemplate, &rootTemplate, rootKey.Public(), rootKey)
	if err != nil {
		return nil, err
	}

	hostTemplate := x509.Certificate{
		IsCA:                  false,
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    x509.SHA1WithRSA,
		PublicKeyAlgorithm:    x509.RSA,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		BasicConstraintsValid: true,
	}

	cert, err := x509.CreateCertificate(rand.Reader, &hostTemplate, &rootTemplate, hostKey.Public(), rootKey)
	if err != nil {
		return nil, err
	}

	caPEM, caPrivPEM, err := getPairPemFormat(caCert, rootKey)

	certPEM, certPrivPEM, err := getPairPemFormat(cert, hostKey)

	deviceTemplate := x509.Certificate{
		IsCA:                  false,
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    x509.SHA1WithRSA,
		PublicKeyAlgorithm:    x509.RSA,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		BasicConstraintsValid: true,
		SubjectKeyId:          []byte("hash"),
	}

	deviceCert, err := x509.CreateCertificate(rand.Reader, &deviceTemplate, &rootTemplate, deviceKey, rootKey)
	if err != nil {
		return nil, err
	}

	deviceCertPEM, err := getPemCertificate(deviceCert)
	if err != nil {
		return nil, err
	}

	record.DeviceCertificate = deviceCertPEM
	record.HostCertificate = certPEM
	record.HostPrivateKey = certPrivPEM
	record.RootCertificate = caPEM
	record.RootPrivateKey = caPrivPEM
	record.SystemBUID = buid
	record.HostID = strings.ToUpper(uuid.NewV4().String())

	return record, nil
}

func LockdownDial(device frames.Device) (*LockdownConnection, error) {
	c, err := Connect(device, LockdownPort)
	if err != nil {
		return nil, err
	}

	s := &Service{conn: MixConnectionClient(c.RawConn)}

	return &LockdownConnection{conn: s, device: device}, nil
}

func (this *LockdownConnection) Pair() (*frames.PairRecord, error) {
	record, err := this.generatePairRecord()
	if err != nil {
		return nil, err
	}

	request := &frames.PairRequest{
		LockdownRequest: *frames.CreateLockdownRequest("Pair"),
		HostName:        frames.ProgramName,
		PairRecord:      record,
		PairingOptions: map[string]interface{}{
			"ExtendedPairingErrors": true,
		},
	}

	if err := this.conn.SendXML(request); err != nil {
		return nil, err
	}
	pkg, err := this.conn.Sync()
	if err != nil {
		return nil, err
	}

	var resp frames.LockdownResponse
	if err := pkg.UnmarshalBody(&resp); err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	return record, nil
}

func (this *LockdownConnection) StopSession() error {
	if this.sslSession == nil {
		return nil
	}

	request := &frames.StopSessionRequest{
		LockdownRequest: *frames.CreateLockdownRequest("StopSession"),
		SessionID:       this.sslSession.SessionID,
	}

	if err := this.conn.SendXML(request); err != nil {
		return err
	}
	pkg, err := this.conn.Sync()
	if err != nil {
		return err
	}

	var resp frames.LockdownResponse
	if err := pkg.UnmarshalBody(&resp); err != nil {
		return err
	}

	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}

	this.sslSession = nil
	return nil
}

func (this *LockdownConnection) StartSession() error {
	if this.sslSession != nil {
		if err := this.StopSession(); err != nil {
			return err
		}
	}

	if this.pairRecord == nil {
		if err := this.Handshake(); err != nil {
			return err
		}
	}

	buid, err := ReadBUID()
	if err != nil {
		return err
	}

	request := &frames.StartSessionRequest{
		LockdownRequest: *frames.CreateLockdownRequest("StartSession"),
		SystemBUID:      buid,
		HostID:          this.pairRecord.HostID,
	}

	if err := this.conn.SendXML(request); err != nil {
		return err
	}
	pkg, err := this.conn.Sync()
	if err != nil {
		return err
	}

	var resp frames.StartSessionResponse

	if err := pkg.UnmarshalBody(&resp); err != nil {
		return err
	}

	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}

	this.sslSession = &resp

	if resp.EnableSessionSSL {
		if err := this.conn.conn.Handshake(this.version, this.pairRecord); err != nil {
			return err
		}
	}

	return nil
}

func (this *LockdownConnection) Handshake() error {
	qtResp, err := this.QueryType()
	if err != nil {
		return err
	}
	if qtResp.Type != "com.apple.mobile.lockdown" {
		return errors.New("queryType not mobile lockdown")
	}

	pvResp, err := this.GetValue("", "ProductVersion")
	if err != nil {
		return err
	}
	if pvResp.Error != "" {
		return fmt.Errorf("%s", pvResp.Error)
	}

	version := strings.Split(pvResp.Value.(string), ".")
	this.version = make([]int, len(version))
	for i, v := range version {
		this.version[i], _ = strconv.Atoi(v)
	}

	resp, err := ReadPairRecord(this.device)
	if err != nil {
		// try pair device
		if _, err := this.Pair(); err != nil {
			return err
		}
		return fmt.Errorf("handshake failed %s", err)
	}
	this.pairRecord = resp

	return nil
}

func (this *LockdownConnection) StartService(service string) (*frames.StartServiceResponse, error) {
	request := &frames.StartServiceRequest{
		LockdownRequest: *frames.CreateLockdownRequest("StartService"),
		Service:         service,
	}

	if this.pairRecord != nil {
		request.EscrowBag = this.pairRecord.EscrowBag
	}

	if err := this.conn.SendXML(request); err != nil {
		return nil, err
	}

	pkg, err := this.conn.Sync()
	if err != nil {
		return nil, err
	}

	var resp frames.StartServiceResponse
	if err := pkg.UnmarshalBody(&resp); err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	return &resp, nil
}

func (this *LockdownConnection) QueryType() (*frames.LockdownTypeResponse, error) {
	req := frames.CreateLockdownRequest("QueryType")
	if err := this.conn.SendXML(req); err != nil {
		return nil, err
	}
	pkg, err := this.conn.Sync()
	if err != nil {
		return nil, err
	}
	var resp frames.LockdownTypeResponse

	if err := pkg.UnmarshalBody(&resp); err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	return &resp, nil
}

func (this *LockdownConnection) GetValue(domain string, key string) (*frames.LockdownValueResponse, error) {
	req := frames.ValueRequest{
		LockdownRequest: *frames.CreateLockdownRequest("GetValue"),
		Domain:          domain,
		Key:             key,
	}

	if err := this.conn.SendXML(req); err != nil {
		return nil, err
	}

	pkg, err := this.conn.Sync()
	if err != nil {
		return nil, err
	}

	var resp frames.LockdownValueResponse
	if err := pkg.UnmarshalBody(&resp); err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	return &resp, nil
}

func (this *LockdownConnection) SetValue(domain string, key string, value interface{}) (*frames.LockdownValueResponse, error) {
	req := frames.ValueRequest{
		LockdownRequest: *frames.CreateLockdownRequest("SetValue"),
		Domain:          domain,
		Key:             key,
		Value:           value,
	}

	if err := this.conn.SendXML(req); err != nil {
		return nil, err
	}
	pkg, err := this.conn.Sync()
	if err != nil {
		return nil, err
	}
	var resp frames.LockdownValueResponse
	if err := pkg.UnmarshalBody(&resp); err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	return &resp, nil
}

func (this *LockdownConnection) GetStringValue(key string) (string, error) {
	resp, err := this.GetValue("", key)

	if err != nil {
		return "", err
	}

	if resp.Error != "" {
		return "", fmt.Errorf("%s", resp.Error)
	}

	return resp.Value.(string), nil
}

func (this *LockdownConnection) EnterRecovery() error {
	req := frames.CreateLockdownRequest("EnterRecovery")

	if err := this.conn.SendXML(req); err != nil {
		return err
	}
	pkg, err := this.conn.Sync()
	if err != nil {
		return err
	}
	var resp frames.LockdownResponse
	if err := pkg.UnmarshalBody(&resp); err != nil {
		return err
	}

	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}

	return nil
}

func (this *LockdownConnection) UniqueDeviceID() (string, error) {
	return this.GetStringValue("UniqueDeviceID")
}

func (this *LockdownConnection) DeviceName() (string, error) {
	return this.GetStringValue("DeviceName")
}

func (this *LockdownConnection) HardwareModel() (string, error) {
	return this.GetStringValue("HardwareModel")
}

func (this *LockdownConnection) DeviceClass() (string, error) {
	return this.GetStringValue("DeviceClass")
}

func (this *LockdownConnection) ProductVersion() (string, error) {
	return this.GetStringValue("ProductVersion")
}

func (this *LockdownConnection) ProductName() (string, error) {
	return this.GetStringValue("ProductName")
}

func (this *LockdownConnection) GenerateConnection(port int, enableSSL bool) (*MixConnection, error) {
	if enableSSL && (this.pairRecord == nil || this.version == nil) {
		if err := this.Handshake(); err != nil {
			return nil, err
		}
	}

	base, err := Connect(this.device, port)
	if err != nil {
		return nil, err
	}

	client := MixConnectionClient(base.RawConn)

	if enableSSL {
		if err := client.Handshake(this.version, this.pairRecord); err != nil {
			return nil, err
		}
	}

	return client, nil
}

func GenerateService(c *MixConnection) *Service {
	return &Service{conn: c}
}

func (this *LockdownConnection) Close() {
	if this.sslSession != nil {
		this.StopSession()
	}

	if this.conn != nil {
		this.conn.conn.Close()
	}

	this.pairRecord = nil
	this.version = nil
}
