package services

import (
	"iconsole/frames"
	"iconsole/tunnel"
)

const (
	MountServiceName             = "com.apple.mobile.mobile_image_mounter"
	ScreenshotServiceName        = "com.apple.mobile.screenshotr"
	SimulateLocationServiceName  = "com.apple.dt.simulatelocation"
	SyslogRelayServiceName       = "com.apple.syslog_relay"
	AFCServiceName               = "com.apple.afc"
	HouseArrestServiceName       = "com.apple.mobile.house_arrest"
	InstallationProxyServiceName = "com.apple.mobile.installation_proxy"
	InstrumentsServiceName       = "com.apple.instruments.remoteserver"
)

// the LockdownConnection must start session
func startService(name string, device frames.Device) (*tunnel.Service, error) {
	lockdown, err := tunnel.LockdownDial(device)
	if err != nil {
		return nil, err
	}
	defer lockdown.Close()

	if err := lockdown.StartSession(); err != nil {
		return nil, err
	}

	dynamicPort, err := lockdown.StartService(name)
	if err != nil {
		return nil, err
	}

	if err := lockdown.StopSession(); err != nil {
		return nil, err
	}

	baseConn, err := lockdown.GenerateConnection(dynamicPort.Port, dynamicPort.EnableServiceSSL)
	if err != nil {
		return nil, err
	}

	return tunnel.GenerateService(baseConn), nil
}

func syncServiceAndCheckError(service *tunnel.Service, resp interface{}) error {
	if pkg, err := service.Sync(); err != nil {
		return err
	} else if err := pkg.UnmarshalBody(resp); err != nil {
		return err
	}
	return nil
}
