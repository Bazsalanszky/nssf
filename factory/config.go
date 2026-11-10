// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * NSSF Configuration Factory
 */

package factory

import (
	"github.com/omec-project/openapi/v2/models"
	utilLogger "github.com/omec-project/util/logger"
)

const (
	NSSF_EXPECTED_CONFIG_VERSION = "1.0.0"
)

type Config struct {
	Info          *Info              `yaml:"info"`
	Configuration *Configuration     `yaml:"configuration"`
	Logger        *utilLogger.Logger `yaml:"logger"`
	CfgLocation   string
	Subscriptions []Subscription `yaml:"subscriptions,omitempty"`
}

type Info struct {
	Version     string `yaml:"version"`
	Description string `yaml:"description,omitempty"`
}

const (
	NSSF_DEFAULT_IPV4     = "127.0.0.31"
	NSSF_DEFAULT_PORT_INT = 8000
)

type Configuration struct {
	NssfName                 string               `yaml:"nssfName,omitempty"`
	Sbi                      *Sbi                 `yaml:"sbi"`
	ServiceNameList          []models.ServiceName `yaml:"serviceNameList"`
	NrfUri                   string               `yaml:"nrfUri"`
	WebuiUri                 string               `yaml:"webuiUri"`
	SupportedNssaiInPlmnList SupportedNssaiInPlmn
	NsiList                  []NsiConfig             `yaml:"nsiList,omitempty"`
	AmfSetList               []AmfSetConfig          `yaml:"amfSetList"`
	AmfList                  []AmfConfig             `yaml:"amfList"`
	TaList                   []TaConfig              `yaml:"taList"`
	MappingListFromPlmn      []MappingFromPlmnConfig `yaml:"mappingListFromPlmn"`
}

type Sbi struct {
	Scheme models.UriScheme `yaml:"scheme"`
	TLS    *TLS             `yaml:"tls"`
	// RegisterIPv4 and RegisterIPv6 are what the NSSF registers at the NRF and
	// uses in its own URIs. Each holds an IPv4 address, an IPv6 address, a host
	// name, a comma-separated list of those, or the name of an environment
	// variable holding one (POD_IP, POD_IPS). Every value goes to the profile
	// field of its own kind, so an IPv6 POD_IP under registerIPv4 is registered
	// in ipv6Addresses.
	RegisterIPv4 string `yaml:"registerIPv4,omitempty"`
	RegisterIPv6 string `yaml:"registerIPv6,omitempty"`
	// BindingAddr is the SBI listen address, of either family, or the name of
	// an environment variable holding it. It wins over BindingIPv4, its legacy
	// alias. The default is 0.0.0.0, which accepts IPv4 and IPv6.
	BindingAddr string `yaml:"bindingAddr,omitempty"`
	BindingIPv4 string `yaml:"bindingIPv4,omitempty"`
	// PreferredFamily (IPv4, the default, or IPv6) picks the address of the
	// NSSF's own URIs when it registers both families and no host name.
	PreferredFamily string `yaml:"preferredFamily,omitempty"`
	Port            int    `yaml:"port"`
}

type TLS struct {
	PEM string `yaml:"pem,omitempty"`
	Key string `yaml:"key,omitempty"`
}

type AmfConfig struct {
	NfId                           string                                  `yaml:"nfId"`
	SupportedNssaiAvailabilityData []models.SupportedNssaiAvailabilityData `yaml:"supportedNssaiAvailabilityData"`
}

type TaConfig struct {
	Tai                  *models.Tai               `yaml:"tai"`
	AccessType           *models.AccessType        `yaml:"accessType"`
	SupportedSnssaiList  []models.Snssai           `yaml:"supportedSnssaiList"`
	RestrictedSnssaiList []models.RestrictedSnssai `yaml:"restrictedSnssaiList,omitempty"`
}

// SnssaiKey is used to avoid using models.Snssai as map key directly due to pointer field issue
type SnssaiKey struct {
	Sst int32
	Sd  string
}

type SupportedNssaiInPlmn map[models.PlmnId]map[SnssaiKey]struct{}

type NsiConfig struct {
	Snssai             *models.Snssai          `yaml:"snssai"`
	NsiInformationList []models.NsiInformation `yaml:"nsiInformationList"`
}

type AmfSetConfig struct {
	AmfSetId                       string                                  `yaml:"amfSetId"`
	AmfList                        []string                                `yaml:"amfList,omitempty"`
	NrfAmfSet                      string                                  `yaml:"nrfAmfSet,omitempty"`
	SupportedNssaiAvailabilityData []models.SupportedNssaiAvailabilityData `yaml:"supportedNssaiAvailabilityData"`
}

type MappingFromPlmnConfig struct {
	OperatorName    string                   `yaml:"operatorName,omitempty"`
	HomePlmnId      *models.PlmnId           `yaml:"homePlmnId"`
	MappingOfSnssai []models.MappingOfSnssai `yaml:"mappingOfSnssai"`
}

type Subscription struct {
	SubscriptionData *models.NssfEventSubscriptionCreateData `yaml:"subscriptionData"`
	SubscriptionId   string                                  `yaml:"subscriptionId"`
}

// Helper function to convert models.Snssai to SnssaiKey
func SnssaiToKey(s models.Snssai) SnssaiKey {
	return SnssaiKey{
		Sst: s.GetSst(),
		Sd:  s.GetSd(),
	}
}

func (c *Config) GetVersion() string {
	if c.Info != nil && c.Info.Version != "" {
		return c.Info.Version
	}
	return ""
}
