// SPDX-FileCopyrightText: 2026 Balázs Toldi
//
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/omec-project/nssf/context"
	"github.com/omec-project/nssf/factory"
	"github.com/omec-project/openapi/v2/models"
)

const (
	testIPv4 = "10.0.0.5"
	testIPv6 = "2001:db8::5"
	testFQDN = "nssf"
)

func TestBuildNFProfileAddresses(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.NSSFContext
		wantFqdn string
		wantIPv4 []string
		wantIPv6 []string
	}{
		{name: "IPv4", ctx: context.NSSFContext{RegisterIPv4: testIPv4}, wantIPv4: []string{testIPv4}},
		{name: "host name goes to fqdn", ctx: context.NSSFContext{RegisterIPv4: testFQDN}, wantFqdn: testFQDN},
		{name: "IPv6 in RegisterIPv4", ctx: context.NSSFContext{RegisterIPv4: testIPv6}, wantIPv6: []string{testIPv6}},
		{name: "IPv6 in RegisterIPv6", ctx: context.NSSFContext{RegisterIPv6: testIPv6}, wantIPv6: []string{testIPv6}},
		{
			name:     "dual stack",
			ctx:      context.NSSFContext{RegisterIPv4: testIPv4, RegisterIPv6: testIPv6},
			wantIPv4: []string{testIPv4},
			wantIPv6: []string{testIPv6},
		},
		{
			name:     "host name and IPv6",
			ctx:      context.NSSFContext{RegisterIPv4: testFQDN, RegisterIPv6: testIPv6},
			wantFqdn: testFQDN,
			wantIPv6: []string{testIPv6},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ctx.NfId = testNfId
			profile, err := getNfProfile(&tt.ctx, nil)
			if err != nil {
				t.Fatalf("getNfProfile() error = %v", err)
			}
			if got := profile.GetFqdn(); got != tt.wantFqdn {
				t.Errorf("fqdn = %q, want %q", got, tt.wantFqdn)
			}
			if got := profile.GetIpv4Addresses(); !reflect.DeepEqual(got, tt.wantIPv4) {
				t.Errorf("ipv4Addresses = %q, want %q", got, tt.wantIPv4)
			}
			if got := profile.GetIpv6Addresses(); !reflect.DeepEqual(got, tt.wantIPv6) {
				t.Errorf("ipv6Addresses = %q, want %q", got, tt.wantIPv6)
			}
		})
	}
}

// TestRegisteredProfileJSON checks the profile the NRF receives, from
// configuration to JSON, for a dual-stack POD_IPS and for the IPv4 and
// host-name configurations used today.
func TestRegisteredProfileJSON(t *testing.T) {
	tests := []struct {
		name string
		sbi  factory.Sbi
		env  string // POD_IPS
		want string // the address part of the profile and of its one service
	}{
		{
			name: "IPv4, as before",
			sbi:  factory.Sbi{RegisterIPv4: testIPv4},
			want: `{"ipv4Addresses":["10.0.0.5"],"service":{"apiPrefix":"http://10.0.0.5:29531",` +
				`"ipEndPoints":[{"ipv4Address":"10.0.0.5","port":29531,"transport":"TCP"}]}}`,
		},
		{
			name: "host name",
			sbi:  factory.Sbi{RegisterIPv4: testFQDN},
			want: `{"fqdn":"nssf","service":{"apiPrefix":"http://nssf:29531","fqdn":"nssf",` +
				`"ipEndPoints":[{"port":29531,"transport":"TCP"}]}}`,
		},
		{
			name: "dual stack from POD_IPS, IPv6 preferred",
			sbi:  factory.Sbi{RegisterIPv4: "POD_IPS", PreferredFamily: "IPv6"},
			env:  testIPv6 + "," + testIPv4,
			want: `{"ipv4Addresses":["10.0.0.5"],"ipv6Addresses":["2001:db8::5"],"service":{"apiPrefix":"http://[2001:db8::5]:29531",` +
				`"ipEndPoints":[{"ipv4Address":"10.0.0.5","port":29531,"transport":"TCP"},` +
				`{"ipv6Address":"2001:db8::5","port":29531,"transport":"TCP"}]}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("POD_IPS", tt.env)
			origCtx, origConfig, origConfigured := *context.NSSF_Self(), factory.NssfConfig, factory.Configured
			t.Cleanup(func() {
				*context.NSSF_Self(), factory.NssfConfig, factory.Configured = origCtx, origConfig, origConfigured
			})
			tt.sbi.Scheme = models.URISCHEME_HTTP
			tt.sbi.Port = 29531
			factory.NssfConfig = factory.Config{Configuration: &factory.Configuration{
				Sbi:             &tt.sbi,
				ServiceNameList: []models.ServiceName{models.SERVICENAME_NNSSF_NSSELECTION},
			}}
			factory.Configured = true
			context.InitNssfContext()

			profile, err := getNfProfile(context.NSSF_Self(), nil)
			if err != nil {
				t.Fatalf("getNfProfile() error = %v", err)
			}
			services := profile.GetNfServices()
			if len(services) != 1 {
				t.Fatalf("nfServices = %+v, want one service", services)
			}
			type serviceAddrs struct {
				ApiPrefix   string              `json:"apiPrefix,omitempty"`
				Fqdn        string              `json:"fqdn,omitempty"`
				IpEndPoints []models.IpEndPoint `json:"ipEndPoints,omitempty"`
			}
			got, err := json.Marshal(struct {
				Fqdn          string       `json:"fqdn,omitempty"`
				Ipv4Addresses []string     `json:"ipv4Addresses,omitempty"`
				Ipv6Addresses []string     `json:"ipv6Addresses,omitempty"`
				Service       serviceAddrs `json:"service"`
			}{
				profile.GetFqdn(), profile.GetIpv4Addresses(), profile.GetIpv6Addresses(),
				serviceAddrs{services[0].GetApiPrefix(), services[0].GetFqdn(), services[0].GetIpEndPoints()},
			})
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("profile addresses =\n  %s\nwant\n  %s", got, tt.want)
			}
		})
	}
}
