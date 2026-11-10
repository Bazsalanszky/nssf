// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * NF Context for NSSF
 *
 * Configuration of NSSF itself shall be accessed with NSSF context
 * Configuration of network slices shall be accessed with configuration factory
 */

package context

import (
	"cmp"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/omec-project/nssf/factory"
	"github.com/omec-project/nssf/logger"
	"github.com/omec-project/openapi/v2/models"
	openapiutils "github.com/omec-project/openapi/v2/utils"
	"github.com/omec-project/util/sbiaddr"
)

var nssfContext = NSSFContext{}

const (
	port           int = 29510
	defaultNrfHost     = "127.0.0.1"
)

const (
	nsSelectionAPIFullVersion        = "2.3.0-alpha.3"
	nsSelectionAPIVersionInURI       = "v2"
	nssaiAvailabilityAPIFullVersion  = "1.3.0-alpha.6"
	nssaiAvailabilityAPIVersionInURI = "v1"
)

// Initialize NSSF context with default value
func init() {
	nssfContext.NfId = uuid.New().String()

	nssfContext.Name = "NSSF"

	nssfContext.UriScheme = models.URISCHEME_HTTPS
	nssfContext.RegisterIPv4 = factory.NSSF_DEFAULT_IPV4
	nssfContext.SBIPort = factory.NSSF_DEFAULT_PORT_INT

	serviceName := []models.ServiceName{
		models.SERVICENAME_NNSSF_NSSELECTION,
		models.SERVICENAME_NNSSF_NSSAIAVAILABILITY,
	}
	nssfContext.NfService = initNfService(serviceName)

	// Until a configuration is loaded the NRF is assumed on the local host, as in
	// InitNssfContext when nrfUri is empty. RegisterIPv4 is the NSSF's own
	// address and says nothing about where the NRF is.
	nssfContext.NrfUri = fmt.Sprintf("%s://%s:%d", models.URISCHEME_HTTPS, defaultNrfHost, port)
}

type NSSFContext struct {
	NfId      string
	Name      string
	UriScheme models.UriScheme
	// RegisterIPv4 is the IPv4 address or the host name registered at the NRF,
	// as before. RegisterIPv6 is the IPv6 address registered next to it.
	RegisterIPv4 string
	RegisterIPv6 string
	// BindingIPv4 is the SBI listen host. Despite the name it may be of either
	// family.
	BindingIPv4   string
	Key           string
	PEM           string
	NfService     map[models.ServiceName]models.NFService
	NrfUri        string
	SBIPort       int
	SbiPreference sbiaddr.Preference
}

// Initialize NSSF context with configuration factory
func InitNssfContext() {
	if !factory.Configured {
		logger.ContextLog.Warnf("NSSF is not configured")
		return
	}
	nssfConfig := factory.NssfConfig

	if nssfConfig.Configuration.NssfName != "" {
		nssfContext.Name = nssfConfig.Configuration.NssfName
	}

	sbi := nssfConfig.Configuration.Sbi
	nssfContext.UriScheme = sbi.Scheme
	initSbiAddrs(&nssfContext, sbi)
	nssfContext.SBIPort = sbi.Port
	if tls := sbi.TLS; tls != nil {
		if tls.Key != "" {
			nssfContext.Key = tls.Key
		}
		if tls.PEM != "" {
			nssfContext.PEM = tls.PEM
		}
	}

	// NF service API versions must track the served SBI routes, not the config schema version.
	nssfContext.NfService = initNfService(nssfConfig.Configuration.ServiceNameList)

	if nssfConfig.Configuration.NrfUri != "" {
		nssfContext.NrfUri = nssfConfig.Configuration.NrfUri
	} else {
		logger.InitLog.Warnln("NRF Uri is empty. Using localhost as NRF IPv4 address")
		nssfContext.NrfUri = fmt.Sprintf("%s://%s:%d", nssfContext.UriScheme, defaultNrfHost, port)
	}
}

// initSbiAddrs sets the register addresses, the family preference and the
// listen host of ctx from the sbi configuration.
func initSbiAddrs(ctx *NSSFContext, sbi *factory.Sbi) {
	addrs, err := sbiaddr.ResolveRegister(sbi.RegisterIPv4, sbi.RegisterIPv6)
	if err != nil {
		logger.ContextLog.Warnf("SBI register address: %v", err)
	}
	if addrs.Empty() {
		logger.ContextLog.Warnf("SBI register address is empty. Using %s as default", factory.NSSF_DEFAULT_IPV4)
		addrs.IPv4 = factory.NSSF_DEFAULT_IPV4
	}
	if addrs.FQDN != "" && addrs.IPv4 != "" {
		logger.ContextLog.Warnf("SBI register address: registering FQDN %s, not IPv4 %s", addrs.FQDN, addrs.IPv4)
	}
	ctx.RegisterIPv4 = cmp.Or(addrs.FQDN, addrs.IPv4)
	ctx.RegisterIPv6 = addrs.IPv6

	if ctx.SbiPreference, err = sbiaddr.ParsePreference(sbi.PreferredFamily); err != nil {
		logger.ContextLog.Warnf("SBI preferredFamily: %v; using IPv4", err)
	}

	bindingSetting := cmp.Or(sbi.BindingAddr, sbi.BindingIPv4)
	if bindingSetting == "" {
		logger.ContextLog.Warnln("error parsing ServerIPv4 address as string. Using the 0.0.0.0 address as default")
	} else if os.Getenv(bindingSetting) != "" {
		logger.ContextLog.Infoln("parsing ServerIPv4 address from ENV Variable")
	}
	if ctx.BindingIPv4, err = sbiaddr.BindHost(sbi.BindingAddr, sbi.BindingIPv4); err != nil {
		logger.ContextLog.Warnf("SBI binding address: %v", err)
	}
}

func initNfService(serviceName []models.ServiceName) (
	nfService map[models.ServiceName]models.NFService,
) {
	nfService = make(map[models.ServiceName]models.NFService)
	addrs := nssfContext.SbiAddrs()
	for idx, name := range serviceName {
		apiFullVersion, apiVersionInURI := getServiceVersion(name)
		service := models.NFService{
			ServiceInstanceId: strconv.Itoa(idx),
			ServiceName:       name,
			Versions: []models.NFServiceVersion{
				{
					ApiFullVersion:  apiFullVersion,
					ApiVersionInUri: apiVersionInURI,
				},
			},
			Scheme:          nssfContext.UriScheme,
			NfServiceStatus: models.NFSERVICESTATUS_REGISTERED,
			ApiPrefix:       new(GetIpv4Uri()),
			IpEndPoints:     openapiutils.NewIpEndPoints(addrs.IPv4, addrs.IPv6, int32(nssfContext.SBIPort)),
		}
		if addrs.FQDN != "" {
			service.SetFqdn(addrs.FQDN)
		}
		nfService[name] = service
	}

	return
}

func getServiceVersion(serviceName models.ServiceName) (apiFullVersion, apiVersionInURI string) {
	switch serviceName {
	case models.SERVICENAME_NNSSF_NSSELECTION:
		return nsSelectionAPIFullVersion, nsSelectionAPIVersionInURI
	case models.SERVICENAME_NNSSF_NSSAIAVAILABILITY:
		return nssaiAvailabilityAPIFullVersion, nssaiAvailabilityAPIVersionInURI
	default:
		return factory.NSSF_EXPECTED_CONFIG_VERSION, "v" + strings.Split(factory.NSSF_EXPECTED_CONFIG_VERSION, ".")[0]
	}
}

// SbiAddrs classifies the register hosts kept in the context: RegisterIPv4
// holds an IPv4 address or a host name, RegisterIPv6 an IPv6 address.
// A value that does not classify is left out; InitNssfContext has already
// warned about it.
func (c *NSSFContext) SbiAddrs() sbiaddr.RegisterAddrs {
	addrs, err := sbiaddr.ParseRegister(c.RegisterIPv4, c.RegisterIPv6)
	if err != nil {
		logger.ContextLog.Debugf("SBI register address: %v", err)
	}
	return addrs
}

// GetSbiUri is the NSSF's own base URI, scheme://host:port, with an IPv6 host
// in brackets. It is "" when no register address is set.
func (c *NSSFContext) GetSbiUri() string {
	return c.SbiAddrs().URI(string(c.UriScheme), c.SBIPort, c.SbiPreference)
}

// GetIpv4Uri is the NSSF's own base URI. Despite the name, the host may be an
// IPv6 address (see GetSbiUri).
func GetIpv4Uri() string {
	return nssfContext.GetSbiUri()
}

func NSSF_Self() *NSSFContext {
	return &nssfContext
}
