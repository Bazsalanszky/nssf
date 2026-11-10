// SPDX-FileCopyrightText: 2026 Balázs Toldi
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/omec-project/nssf/factory"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/util/sbiaddr"
)

const (
	testPort = 29531
	testIPv4 = "10.0.0.5"
	testIPv6 = "2001:db8::5"
	testFQDN = "nssf"
	podIP    = "POD_IP"
	podIPs   = "POD_IPS"

	uriFQDN    = "http://nssf:29531"
	uriIPv4    = "http://10.0.0.5:29531"
	uriIPv6    = "http://[2001:db8::5]:29531"
	uriDefault = "http://127.0.0.31:29531"
)

// Before a configuration is loaded, the NRF URI points at the local host,
// like the nrfUri default in InitNssfContext. It used to be built from the
// NSSF's own default register address (https://127.0.0.31:29510).
func TestDefaultNrfUri(t *testing.T) {
	if got, want := NSSF_Self().NrfUri, "https://127.0.0.1:29510"; got != want {
		t.Errorf("default NrfUri = %q, want %q", got, want)
	}
}

// initWith runs InitNssfContext with sbi and returns the resulting context.
// The environment holds only env (POD_IP and POD_IPS are cleared otherwise).
// The package and factory state is restored when the test ends.
func initWith(t *testing.T, sbi factory.Sbi, env map[string]string) NSSFContext {
	t.Helper()
	t.Setenv(podIP, "")
	t.Setenv(podIPs, "")
	for k, v := range env {
		t.Setenv(k, v)
	}
	origCtx, origConfig, origConfigured := nssfContext, factory.NssfConfig, factory.Configured
	t.Cleanup(func() {
		nssfContext, factory.NssfConfig, factory.Configured = origCtx, origConfig, origConfigured
	})
	factory.NssfConfig = factory.Config{Configuration: &factory.Configuration{
		Sbi:             &sbi,
		ServiceNameList: []models.ServiceName{models.SERVICENAME_NNSSF_NSSELECTION},
		NrfUri:          "http://nrf:29510",
	}}
	factory.Configured = true
	InitNssfContext()
	return nssfContext
}

func ipv4EndPoint(addr string) models.IpEndPoint {
	ep := models.NewIpEndPoint()
	ep.SetIpv4Address(addr)
	ep.SetTransport(models.TRANSPORTPROTOCOL_TCP)
	ep.SetPort(testPort)
	return *ep
}

func ipv6EndPoint() models.IpEndPoint {
	ep := models.NewIpEndPoint()
	ep.SetIpv6Address(testIPv6)
	ep.SetTransport(models.TRANSPORTPROTOCOL_TCP)
	ep.SetPort(testPort)
	return *ep
}

func portEndPoint() models.IpEndPoint {
	ep := models.NewIpEndPoint()
	ep.SetTransport(models.TRANSPORTPROTOCOL_TCP)
	ep.SetPort(testPort)
	return *ep
}

func TestInitNssfContextRegisterAddrs(t *testing.T) {
	tests := []struct {
		env           map[string]string
		name          string
		sbi           factory.Sbi
		wantIPv4      string // context RegisterIPv4: IPv4 address or host name
		wantIPv6      string
		wantURI       string
		wantFqdn      string // service fqdn
		wantEndPoints []models.IpEndPoint
		wantPref      sbiaddr.Preference
	}{
		{
			name:          "chart default host name",
			sbi:           factory.Sbi{RegisterIPv4: testFQDN},
			wantIPv4:      testFQDN,
			wantURI:       uriFQDN,
			wantFqdn:      testFQDN,
			wantEndPoints: []models.IpEndPoint{portEndPoint()},
		},
		{
			name:          "IPv4 literal",
			sbi:           factory.Sbi{RegisterIPv4: testIPv4},
			wantIPv4:      testIPv4,
			wantURI:       uriIPv4,
			wantEndPoints: []models.IpEndPoint{ipv4EndPoint(testIPv4)},
		},
		{
			name:          "IPv6 literal under registerIPv4",
			sbi:           factory.Sbi{RegisterIPv4: testIPv6},
			wantIPv6:      testIPv6,
			wantURI:       uriIPv6,
			wantEndPoints: []models.IpEndPoint{ipv6EndPoint()},
		},
		{
			name:          "IPv6 under registerIPv6, canonicalised",
			sbi:           factory.Sbi{RegisterIPv6: "[2001:DB8:0::5]"},
			wantIPv6:      testIPv6,
			wantURI:       uriIPv6,
			wantEndPoints: []models.IpEndPoint{ipv6EndPoint()},
		},
		{
			name:          "dual stack, IPv4 preferred by default",
			sbi:           factory.Sbi{RegisterIPv4: testIPv4, RegisterIPv6: testIPv6},
			wantIPv4:      testIPv4,
			wantIPv6:      testIPv6,
			wantURI:       uriIPv4,
			wantEndPoints: []models.IpEndPoint{ipv4EndPoint(testIPv4), ipv6EndPoint()},
		},
		{
			name:          "dual stack, preferredFamily IPv6",
			sbi:           factory.Sbi{RegisterIPv4: testIPv4, RegisterIPv6: testIPv6, PreferredFamily: "IPv6"},
			wantIPv4:      testIPv4,
			wantIPv6:      testIPv6,
			wantURI:       uriIPv6,
			wantEndPoints: []models.IpEndPoint{ipv4EndPoint(testIPv4), ipv6EndPoint()},
			wantPref:      sbiaddr.PreferIPv6,
		},
		{
			name:          "unknown preferredFamily falls back to IPv4",
			sbi:           factory.Sbi{RegisterIPv4: testIPv4, RegisterIPv6: testIPv6, PreferredFamily: "both"},
			wantIPv4:      testIPv4,
			wantIPv6:      testIPv6,
			wantURI:       uriIPv4,
			wantEndPoints: []models.IpEndPoint{ipv4EndPoint(testIPv4), ipv6EndPoint()},
		},
		{
			name:          "POD_IP holding IPv4",
			sbi:           factory.Sbi{RegisterIPv4: podIP},
			env:           map[string]string{podIP: testIPv4},
			wantIPv4:      testIPv4,
			wantURI:       uriIPv4,
			wantEndPoints: []models.IpEndPoint{ipv4EndPoint(testIPv4)},
		},
		{
			name:          "POD_IP holding IPv6 under registerIPv4",
			sbi:           factory.Sbi{RegisterIPv4: podIP},
			env:           map[string]string{podIP: testIPv6},
			wantIPv6:      testIPv6,
			wantURI:       uriIPv6,
			wantEndPoints: []models.IpEndPoint{ipv6EndPoint()},
		},
		{
			name:          "POD_IPS dual stack",
			sbi:           factory.Sbi{RegisterIPv4: podIPs},
			env:           map[string]string{podIPs: testIPv6 + "," + testIPv4},
			wantIPv4:      testIPv4,
			wantIPv6:      testIPv6,
			wantURI:       uriIPv4,
			wantEndPoints: []models.IpEndPoint{ipv4EndPoint(testIPv4), ipv6EndPoint()},
		},
		{
			name:          "host name plus registerIPv6",
			sbi:           factory.Sbi{RegisterIPv4: testFQDN, RegisterIPv6: podIP},
			env:           map[string]string{podIP: testIPv6},
			wantIPv4:      testFQDN,
			wantIPv6:      testIPv6,
			wantURI:       uriFQDN,
			wantFqdn:      testFQDN,
			wantEndPoints: []models.IpEndPoint{ipv6EndPoint()},
		},
		{
			name:          "host name wins over IPv4",
			sbi:           factory.Sbi{RegisterIPv4: testFQDN + "," + testIPv4},
			wantIPv4:      testFQDN,
			wantURI:       uriFQDN,
			wantFqdn:      testFQDN,
			wantEndPoints: []models.IpEndPoint{portEndPoint()},
		},
		{
			name:          "empty uses the default",
			sbi:           factory.Sbi{},
			wantIPv4:      factory.NSSF_DEFAULT_IPV4,
			wantURI:       uriDefault,
			wantEndPoints: []models.IpEndPoint{ipv4EndPoint(factory.NSSF_DEFAULT_IPV4)},
		},
		{
			name:          "unset POD_IP uses the default",
			sbi:           factory.Sbi{RegisterIPv4: podIP},
			wantIPv4:      factory.NSSF_DEFAULT_IPV4,
			wantURI:       uriDefault,
			wantEndPoints: []models.IpEndPoint{ipv4EndPoint(factory.NSSF_DEFAULT_IPV4)},
		},
		{
			name:          "invalid value uses the default",
			sbi:           factory.Sbi{RegisterIPv4: "10.0.0.256"},
			wantIPv4:      factory.NSSF_DEFAULT_IPV4,
			wantURI:       uriDefault,
			wantEndPoints: []models.IpEndPoint{ipv4EndPoint(factory.NSSF_DEFAULT_IPV4)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.sbi.Scheme = models.URISCHEME_HTTP
			tt.sbi.Port = testPort
			ctx := initWith(t, tt.sbi, tt.env)

			if ctx.RegisterIPv4 != tt.wantIPv4 || ctx.RegisterIPv6 != tt.wantIPv6 {
				t.Errorf("RegisterIPv4, RegisterIPv6 = %q, %q, want %q, %q",
					ctx.RegisterIPv4, ctx.RegisterIPv6, tt.wantIPv4, tt.wantIPv6)
			}
			if ctx.SbiPreference != tt.wantPref {
				t.Errorf("SbiPreference = %v, want %v", ctx.SbiPreference, tt.wantPref)
			}
			if got := ctx.GetSbiUri(); got != tt.wantURI {
				t.Errorf("GetSbiUri() = %q, want %q", got, tt.wantURI)
			}
			if got := GetIpv4Uri(); got != tt.wantURI {
				t.Errorf("GetIpv4Uri() = %q, want %q", got, tt.wantURI)
			}
			if u, err := url.Parse(tt.wantURI); err != nil || u.Port() == "" {
				t.Errorf("url.Parse(%q) = %v, %v: want a URI with a port", tt.wantURI, u, err)
			}

			service, ok := ctx.NfService[models.SERVICENAME_NNSSF_NSSELECTION]
			if !ok {
				t.Fatalf("NfService has no %s: %v", models.SERVICENAME_NNSSF_NSSELECTION, ctx.NfService)
			}
			if got := service.GetApiPrefix(); got != tt.wantURI {
				t.Errorf("service apiPrefix = %q, want %q", got, tt.wantURI)
			}
			if got := service.GetFqdn(); got != tt.wantFqdn {
				t.Errorf("service fqdn = %q, want %q", got, tt.wantFqdn)
			}
			if got := service.GetIpEndPoints(); !reflect.DeepEqual(got, tt.wantEndPoints) {
				t.Errorf("service ipEndPoints = %+v, want %+v", got, tt.wantEndPoints)
			}
		})
	}
}

func TestInitNssfContextBindingAddr(t *testing.T) {
	tests := []struct {
		env  map[string]string
		name string
		sbi  factory.Sbi
		want string
	}{
		{name: "default", want: sbiaddr.DefaultBindHost},
		{name: "bindingIPv4 alone, as before", sbi: factory.Sbi{BindingIPv4: testIPv4}, want: testIPv4},
		{name: "bindingAddr wins over bindingIPv4", sbi: factory.Sbi{BindingAddr: "::", BindingIPv4: "0.0.0.0"}, want: "::"},
		{name: "bindingAddr alone", sbi: factory.Sbi{BindingAddr: testIPv6}, want: testIPv6},
		{name: "bracketed IPv6 is unwrapped", sbi: factory.Sbi{BindingIPv4: "[::]"}, want: "::"},
		{
			name: "bindingIPv4 POD_IP holding IPv6",
			sbi:  factory.Sbi{BindingIPv4: podIP},
			env:  map[string]string{podIP: testIPv6},
			want: testIPv6,
		},
		{
			name: "bindingAddr POD_IP holding IPv4",
			sbi:  factory.Sbi{BindingAddr: podIP},
			env:  map[string]string{podIP: testIPv4},
			want: testIPv4,
		},
		{
			name: "a list cannot be bound: the dual-stack wildcard",
			sbi:  factory.Sbi{BindingAddr: podIPs},
			env:  map[string]string{podIPs: testIPv4 + "," + testIPv6},
			want: sbiaddr.DefaultBindHost,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.sbi.Scheme = models.URISCHEME_HTTP
			tt.sbi.Port = testPort
			if got := initWith(t, tt.sbi, tt.env).BindingIPv4; got != tt.want {
				t.Errorf("BindingIPv4 = %q, want %q", got, tt.want)
			}
		})
	}
}

// Code and tests that set the context fields directly get the same
// classification as InitNssfContext.
func TestSbiAddrsFromContextFields(t *testing.T) {
	tests := []struct {
		name string
		ctx  NSSFContext
		want sbiaddr.RegisterAddrs
		uri  string
	}{
		{name: "empty", ctx: NSSFContext{}, uri: ""},
		{
			name: "IPv4",
			ctx:  NSSFContext{RegisterIPv4: "127.0.0.42", UriScheme: models.URISCHEME_HTTPS, SBIPort: 8000},
			want: sbiaddr.RegisterAddrs{IPv4: "127.0.0.42"},
			uri:  "https://127.0.0.42:8000",
		},
		{
			name: "IPv6 put in RegisterIPv4",
			ctx:  NSSFContext{RegisterIPv4: testIPv6, UriScheme: models.URISCHEME_HTTPS, SBIPort: 8000},
			want: sbiaddr.RegisterAddrs{IPv6: testIPv6},
			uri:  "https://[2001:db8::5]:8000",
		},
		{
			name: "host name and IPv6, IPv6 preferred: the host name still wins",
			ctx: NSSFContext{
				RegisterIPv4: testFQDN, RegisterIPv6: testIPv6, UriScheme: models.URISCHEME_HTTPS, SBIPort: 8000,
				SbiPreference: sbiaddr.PreferIPv6,
			},
			want: sbiaddr.RegisterAddrs{FQDN: testFQDN, IPv6: testIPv6},
			uri:  "https://nssf:8000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ctx.SbiAddrs(); got != tt.want {
				t.Errorf("SbiAddrs() = %+v, want %+v", got, tt.want)
			}
			if got := tt.ctx.GetSbiUri(); got != tt.uri {
				t.Errorf("GetSbiUri() = %q, want %q", got, tt.uri)
			}
		})
	}
}
