package services

import (
	"errors"
	"fmt"
	"iconsole/frames"
	"iconsole/tunnel"
)

type ApplicationType string

const (
	System   = ApplicationType("System")
	User     = ApplicationType("User")
	Internal = ApplicationType("Internal")
	Any      = ApplicationType("Any")
)

type baseCommand struct {
	Command       string                   `plist:"Command"`
	ClientOptions *InstallationProxyOption `plist:"ClientOptions,omitempty"`
}

type InstallationProxyOption struct {
	ApplicationType  ApplicationType `plist:"ApplicationType,omitempty"`
	ReturnAttributes []string        `plist:"ReturnAttributes,omitempty"`
	MetaData         bool            `plist:"com.apple.mobile_installation.metadata,omitempty"`
	BundleIDs        []string        `plist:"BundleIDs,omitempty"` /* for Lookup */
}

type InstallationProxyService struct {
	service *tunnel.Service
}

func NewInstallationProxyService(device frames.Device) (*InstallationProxyService, error) {
	serv, err := startService(InstallationProxyServiceName, device)
	if err != nil {
		return nil, err
	}

	return &InstallationProxyService{service: serv}, nil
}

func (this *InstallationProxyService) Browse(opt *InstallationProxyOption) ([]map[string]interface{}, error) {
	m := baseCommand{
		Command:       "Browse",
		ClientOptions: opt,
	}

	if err := this.service.SendXML(m); err != nil {
		return nil, err
	}

	var apps []map[string]interface{}

	for {
		if pkg, err := this.service.Sync(); err != nil {
			return nil, err
		} else {
			r := map[string]interface{}{}
			if err := pkg.UnmarshalBody(&r); err != nil {
				return nil, err
			}
			if r["Status"].(string) == "Complete" {
				break
			} else if l, ok := r["CurrentList"].([]interface{}); ok {
				for _, v := range l {
					apps = append(apps, v.(map[string]interface{}))
				}
			}
		}
	}

	return apps, nil
}

func (this *InstallationProxyService) Lookup(opt *InstallationProxyOption) (map[string]interface{}, error) {
	req := baseCommand{
		Command:       "Lookup",
		ClientOptions: opt,
	}

	if err := this.service.SendXML(req); err != nil {
		return nil, err
	}

	if pkg, err := this.service.Sync(); err != nil {
		return nil, err
	} else {
		var r map[string]interface{}
		if err := pkg.UnmarshalBody(&r); err != nil {
			return nil, err
		}

		status, ok := r["Status"].(string)
		if !ok {
			return nil, errors.New("wrap object failed")
		}

		if status == "Complete" {
			if res, ok := r["LookupResult"].(map[string]interface{}); ok {
				return res, nil
			} else {
				return nil, errors.New("wrap object failed")
			}
		} else {
			return nil, fmt.Errorf("status: %s", status)
		}
	}
}
