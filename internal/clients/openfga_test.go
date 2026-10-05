package clients

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const (
	plainURL = "http://openfga:8080"
	tlsURL   = "https://openfga.example.org"
	secret   = "t0ps3cr3t11"
)

func TestParseCredentials(t *testing.T) {
	got, err := parseCredentials([]byte(`{"api_url":"http://openfga:8080","api_token":"t0ps3cr3t11"}`))
	if err != nil {
		t.Fatalf("parseCredentials: %v", err)
	}
	want := Credentials{APIURL: plainURL, APIToken: secret}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("credentials: %s", diff)
	}
}

func TestParseCredentialsRejectsUnknownKeys(t *testing.T) {
	// A key we silently ignored would be worse than an error: a misspelled
	// api_token reads as an unauthenticated server, and an option the OpenFGA
	// Terraform provider has no equivalent for reads as supported.
	for name, doc := range map[string]string{
		"Misspelled":    `{"api_url":"http://openfga:8080","apitoken":"secret"}`,
		"Unsupported":   `{"api_url":"http://openfga:8080","insecure_skip_verify":true}`,
		"IncusLeftover": `{"address":"http://openfga:8080"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseCredentials([]byte(doc)); err == nil {
				t.Error("expected an error, got none")
			}
		})
	}
}

func TestTerraformConfiguration(t *testing.T) {
	for name, tc := range map[string]struct {
		creds Credentials
		want  map[string]any
	}{
		"NoAuthentication": {
			creds: Credentials{APIURL: plainURL},
			want:  map[string]any{"api_url": plainURL},
		},
		"PresharedKey": {
			creds: Credentials{APIURL: plainURL, APIToken: secret},
			want: map[string]any{
				"api_url":   plainURL,
				"api_token": secret,
			},
		},
		"ClientCredentials": {
			creds: Credentials{
				APIURL:         tlsURL,
				ClientID:       "crossplane",
				ClientSecret:   secret,
				APITokenIssuer: "https://sso.example.org",
				APIAudience:    tlsURL,
				APIScopes:      "read write",
			},
			want: map[string]any{
				"api_url":          tlsURL,
				"client_id":        "crossplane",
				"client_secret":    secret,
				"api_token_issuer": "https://sso.example.org",
				"api_audience":     tlsURL,
				"api_scopes":       "read write",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := tc.creds.terraformConfiguration(t.TempDir())
			if err != nil {
				t.Fatalf("terraformConfiguration: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("configuration: %s", diff)
			}
		})
	}
}

func TestTerraformConfigurationErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		creds   Credentials
		wantErr string
	}{
		"NoURL": {
			creds:   Credentials{APIToken: secret},
			wantErr: "api_url is required",
		},
		"TokenAndClientCredentials": {
			creds: Credentials{
				APIURL:         plainURL,
				APIToken:       secret,
				ClientID:       "crossplane",
				ClientSecret:   secret,
				APITokenIssuer: "https://sso.example.org",
			},
			wantErr: "mutually exclusive",
		},
		"PartialClientCredentials": {
			creds: Credentials{
				APIURL:   plainURL,
				ClientID: "crossplane",
			},
			wantErr: "api_token_issuer, client_secret",
		},
		"NotAPEMAuthority": {
			creds: Credentials{
				APIURL: tlsURL,
				CACert: "not a certificate",
			},
			wantErr: "PEM",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := tc.creds.terraformConfiguration(t.TempDir())
			if err == nil {
				t.Fatal("expected an error, got none")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not mention %q", err, tc.wantErr)
			}
		})
	}
}

// testCA is a throwaway self-signed certificate, only ever parsed as PEM.
const testCA = `-----BEGIN CERTIFICATE-----
MIIBITAdBgNVBAMTFmNyb3NzcGxhbmUtdGVzdC1jYS1vbmx5MA0GCSqGSIb3DQEB
CwUAA0EAqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq
qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq
-----END CERTIFICATE-----`

const testCA2 = `-----BEGIN CERTIFICATE-----
MIIBITAdBgNVBAMTFmNyb3NzcGxhbmUtdGVzdC1jYS10d28wDQYJKoZIhvcNAQEL
BQADQQCrq6urq6urq6urq6urq6urq6urq6urq6urq6urq6urq6urq6urq6urq6ur
q6urq6urq6urq6urq6urq6urq6urq6urq6ur
-----END CERTIFICATE-----`

// TestTrustCA checks that authorities compose instead of overwriting each
// other, since the trust store belongs to the provider pod rather than to a
// single ProviderConfig.
func TestTrustCA(t *testing.T) {
	t.Setenv("SSL_CERT_FILE", "")
	caTrust.Lock()
	caTrust.added = map[string]struct{}{}
	caTrust.path = ""
	caTrust.Unlock()
	t.Cleanup(func() {
		caTrust.Lock()
		caTrust.added = map[string]struct{}{}
		caTrust.path = ""
		caTrust.Unlock()
	})

	dir := t.TempDir()
	for _, cert := range []string{testCA, testCA2, testCA} {
		if err := trustCA(dir, cert); err != nil {
			t.Fatalf("trustCA: %v", err)
		}
	}

	bundle := os.Getenv("SSL_CERT_FILE")
	if bundle == "" {
		t.Fatal("SSL_CERT_FILE was not set")
	}
	b, err := os.ReadFile(bundle) //nolint:gosec // path comes from trustCA
	if err != nil {
		t.Fatalf("cannot read the bundle: %v", err)
	}
	got := string(b)

	// Both authorities are in, each exactly once despite the repeat.
	for _, cert := range []string{testCA, testCA2} {
		if n := strings.Count(got, strings.SplitN(cert, "\n", 2)[1]); n != 1 {
			t.Errorf("certificate appears %d times, want 1", n)
		}
	}
	// The system authorities, when the host has any, must survive: pointing
	// SSL_CERT_FILE at a bundle replaces the system trust store rather than
	// adding to it.
	if sys := systemCertificates(); len(sys) > 0 && !strings.HasPrefix(got, string(sys)) {
		t.Error("the bundle does not start with the system certificates")
	}
}
