package tlsprofile

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
)

func TestApplyToHTTPServingInfo(t *testing.T) {
	t.Run("intermediate", func(t *testing.T) {
		spec, err := EffectiveSpec(&configv1.TLSSecurityProfile{Type: configv1.TLSProfileIntermediateType})
		if err != nil {
			t.Fatal(err)
		}
		serving := &configv1.HTTPServingInfo{}
		if err := ApplyToHTTPServingInfo(serving, spec); err != nil {
			t.Fatal(err)
		}
		if serving.MinTLSVersion != string(configv1.VersionTLS12) {
			t.Fatalf("min version: %q", serving.MinTLSVersion)
		}
		if len(serving.CipherSuites) == 0 {
			t.Fatal("expected ciphers")
		}
	})

	t.Run("modern tls13 keeps non-empty ciphers to block defaults", func(t *testing.T) {
		spec, err := EffectiveSpec(&configv1.TLSSecurityProfile{Type: configv1.TLSProfileModernType})
		if err != nil {
			t.Fatal(err)
		}
		serving := &configv1.HTTPServingInfo{}
		if err := ApplyToHTTPServingInfo(serving, spec); err != nil {
			t.Fatal(err)
		}
		if serving.MinTLSVersion != string(configv1.VersionTLS13) {
			t.Fatalf("min version: %q", serving.MinTLSVersion)
		}
		if len(serving.CipherSuites) == 0 {
			t.Fatal("expected non-empty cipher list so library-go defaults are not reapplied")
		}
	})

	t.Run("nil serving", func(t *testing.T) {
		if err := ApplyToHTTPServingInfo(nil, &configv1.TLSProfileSpec{MinTLSVersion: configv1.VersionTLS12}); err == nil {
			t.Fatal("expected error")
		}
	})
}
