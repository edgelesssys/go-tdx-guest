// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pcs

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestPckCrlURL(t *testing.T) {
	want := SgxBaseURL + "/pckcrl?ca=platform&encoding=der"

	if got := PckCrlURL("platform"); got != want {
		t.Errorf(`PckCrlURL("platform") = %q. Expected %q`, got, want)
	}
}

func TestTcbInfoURL(t *testing.T) {
	want := TdxBaseURL + "/tcb?fmspc=50806f000000"
	fmspcBytes := []byte{80, 128, 111, 0, 0, 0}
	fmspc := hex.EncodeToString(fmspcBytes)
	if got := TcbInfoURL(fmspc); got != want {
		t.Errorf("TcbInfoURL(%q) = %q. Expected %q", fmspc, got, want)
	}
}

func TestQeIdentityURL(t *testing.T) {
	want := TdxBaseURL + "/qe/identity"
	if got := QeIdentityURL(); got != want {
		t.Errorf("QEIdentityURL() = %q. Expected %q", got, want)
	}
}

func createExtension(t *testing.T, oid asn1.ObjectIdentifier, value []byte) pkix.Extension {
	t.Helper()
	valueBytes, err := asn1.Marshal(value)
	if err != nil {
		t.Fatalf("Failed to marshal value %v: %v", value, err)
	}

	return pkix.Extension{
		Id:    oid,
		Value: valueBytes,
	}
}

func testPCKCert(t *testing.T, exts PckExtensions) *x509.Certificate {
	t.Helper()

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	ppid, err := hex.DecodeString(exts.PPID)
	if err != nil {
		t.Fatalf("Failed to decode PPID: %v", err)
	}
	pceid, err := hex.DecodeString(exts.PCEID)
	if err != nil {
		t.Fatalf("Failed to decode PCEID: %v", err)
	}
	fmspc, err := hex.DecodeString(exts.FMSPC)
	if err != nil {
		t.Fatalf("Failed to decode FMSPC: %v", err)
	}
	piid, err := hex.DecodeString(exts.PIID)
	if err != nil {
		t.Fatalf("Failed to decode PIID: %v", err)
	}

	// Create nested SGX extensions.
	sgxExtensions := []pkix.Extension{
		// Minimum of 4 SGX extensions required.
		createExtension(t, OidPPID, ppid),
		createExtension(t, OidPCEID, pceid),
		createExtension(t, OidFMSPC, fmspc),
		createExtension(t, OidPIID, piid),
	}

	sgxPayload, err := asn1.Marshal(sgxExtensions)
	if err != nil {
		t.Fatalf("Error marshaling SGX extensions: %v", err)
	}

	extensions := []pkix.Extension{{Id: OidSgxExtension, Value: sgxPayload}}

	// 6 total top-level extensions are required.
	// KeyUsage and BasicConstraints extensions are automatically added by x509.CreateCertificate, so we add 3 more to reach the required total.
	for i := 1; i <= 3; i++ {
		fakeoid := asn1.ObjectIdentifier([]int{1, 2, 840, i})
		fakePayload := []byte("fake extension")

		extensions = append(extensions, pkix.Extension{Id: fakeoid, Value: fakePayload})
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		ExtraExtensions:       extensions,
	}

	pckBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if err != nil {
		t.Fatalf("Failed to create test PCK certificate: %v", err)
	}

	parsed, err := x509.ParseCertificate(pckBytes)
	if err != nil {
		t.Fatalf("Failed to parse generated PCK certificate: %v", err)
	}

	return parsed
}

func TestParsePckCertExtension(t *testing.T) {
	expectedExts := PckExtensions{
		PPID:  hex.EncodeToString(bytes.Repeat([]byte{0x41}, ppidSize)),
		PCEID: hex.EncodeToString(bytes.Repeat([]byte{0x42}, pceIDSize)),
		FMSPC: hex.EncodeToString(bytes.Repeat([]byte{0x43}, fmspcSize)),
		PIID:  hex.EncodeToString(bytes.Repeat([]byte{0x44}, piidSize)),
	}

	cert := testPCKCert(t, expectedExts)

	exts, err := PckCertificateExtensions(cert)
	if err != nil {
		t.Fatalf("ParsePckCertExtension() error = %v", err)
	}

	if !cmp.Equal(*exts, expectedExts) {
		t.Errorf("ParsePckCertExtension() = %v, want %v", *exts, expectedExts)
	}
}
