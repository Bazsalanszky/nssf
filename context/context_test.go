// SPDX-FileCopyrightText: 2026 Balázs Toldi
//
// SPDX-License-Identifier: Apache-2.0

package context

import "testing"

// Before a configuration is loaded, the NRF URI points at the local host,
// like the nrfUri default in InitNssfContext. It used to be built from the
// NSSF's own default register address (https://127.0.0.31:29510).
func TestDefaultNrfUri(t *testing.T) {
	if got, want := NSSF_Self().NrfUri, "https://127.0.0.1:29510"; got != want {
		t.Errorf("default NrfUri = %q, want %q", got, want)
	}
}
