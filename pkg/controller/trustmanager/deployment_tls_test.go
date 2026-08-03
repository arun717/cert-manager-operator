package trustmanager

import (
	"context"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/openshift/cert-manager-operator/pkg/controller/common/fakes"
	"github.com/openshift/cert-manager-operator/pkg/tlsprofile"
)

func TestApplyTrustManagerWebhookTLSArgs(t *testing.T) {
	tests := []struct {
		name       string
		spec       *configv1.TLSProfileSpec
		wantKeys   []string
		wantAbsent []string
	}{
		{
			name: "intermediate sets min version and ciphers",
			spec: &configv1.TLSProfileSpec{
				Ciphers:       []string{"ECDHE-RSA-AES128-GCM-SHA256"},
				MinTLSVersion: configv1.VersionTLS12,
			},
			wantKeys: []string{"--tls-min-version", "--tls-cipher-suites"},
		},
		{
			name: "modern tls13 omits cipher suites",
			spec: &configv1.TLSProfileSpec{
				Ciphers:       []string{"TLS_AES_128_GCM_SHA256"},
				MinTLSVersion: configv1.VersionTLS13,
			},
			wantKeys:   []string{"--tls-min-version"},
			wantAbsent: []string{"--tls-cipher-suites"},
		},
		{
			name: "nil spec is no-op",
			spec: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: trustManagerDeploymentName, Namespace: operandNamespace},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name: trustManagerContainerName,
								Args: []string{"--webhook-port=6443", "--tls-cipher-suites=STALE"},
							}},
						},
					},
				},
			}
			if err := applyTrustManagerWebhookTLSArgs(dep, tt.spec); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			argMap := map[string]string{}
			for _, a := range dep.Spec.Template.Spec.Containers[0].Args {
				parts := strings.SplitN(a, "=", 2)
				if len(parts) == 2 {
					argMap[parts[0]] = parts[1]
				} else {
					argMap[parts[0]] = ""
				}
			}
			for _, key := range tt.wantKeys {
				if _, ok := argMap[key]; !ok {
					t.Fatalf("expected arg key %q, got %#v", key, argMap)
				}
			}
			for _, key := range tt.wantAbsent {
				if _, ok := argMap[key]; ok {
					t.Fatalf("did not expect arg key %q, got %#v", key, argMap)
				}
			}
			if tt.spec != nil && tt.spec.MinTLSVersion == configv1.VersionTLS13 {
				if argMap["--tls-min-version"] != "VersionTLS13" {
					t.Fatalf("got min version %q", argMap["--tls-min-version"])
				}
			}
		})
	}
}

func TestApplyClusterTLSProfile_adherence(t *testing.T) {
	tests := []struct {
		name          string
		apiServer     *configv1.APIServer
		wantTLSArgs   bool
		wantMinVer    string
		wantCipherKey bool
	}{
		{
			name: "strict modern injects tls13 min version without ciphers",
			apiServer: &configv1.APIServer{
				ObjectMeta: metav1.ObjectMeta{Name: apiServerClusterName},
				Spec: configv1.APIServerSpec{
					TLSAdherence: configv1.TLSAdherencePolicyStrictAllComponents,
					TLSSecurityProfile: &configv1.TLSSecurityProfile{
						Type: configv1.TLSProfileModernType,
					},
				},
			},
			wantTLSArgs:   true,
			wantMinVer:    "VersionTLS13",
			wantCipherKey: false,
		},
		{
			name: "legacy adherence skips injection",
			apiServer: &configv1.APIServer{
				ObjectMeta: metav1.ObjectMeta{Name: apiServerClusterName},
				Spec: configv1.APIServerSpec{
					TLSAdherence: configv1.TLSAdherencePolicyLegacyAdheringComponentsOnly,
					TLSSecurityProfile: &configv1.TLSSecurityProfile{
						Type: configv1.TLSProfileModernType,
					},
				},
			},
			wantTLSArgs: false,
		},
		{
			name:        "missing apiserver skips injection",
			apiServer:   nil,
			wantTLSArgs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(trustManagerImageNameEnvVarName, testImage)
			r := testReconciler(t)
			r.CtrlClient = fakeCtrlClientWithAPIServer(tt.apiServer)

			tm := testTrustManager().Build()
			dep, err := r.getDeploymentObject(tm, getResourceLabels(tm), getResourceAnnotations(tm), "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			argMap := containerArgMap(dep)
			_, hasMin := argMap["--tls-min-version"]
			_, hasCipher := argMap["--tls-cipher-suites"]
			if tt.wantTLSArgs != hasMin {
				t.Fatalf("wantTLSArgs=%v hasMin=%v args=%#v", tt.wantTLSArgs, hasMin, argMap)
			}
			if tt.wantTLSArgs && argMap["--tls-min-version"] != tt.wantMinVer {
				t.Fatalf("min version got %q want %q", argMap["--tls-min-version"], tt.wantMinVer)
			}
			if hasCipher != tt.wantCipherKey {
				t.Fatalf("wantCipherKey=%v hasCipher=%v", tt.wantCipherKey, hasCipher)
			}
		})
	}
}

func TestApplyClusterTLSProfile_intermediateCiphers(t *testing.T) {
	t.Setenv(trustManagerImageNameEnvVarName, testImage)
	apiServer := &configv1.APIServer{
		ObjectMeta: metav1.ObjectMeta{Name: apiServerClusterName},
		Spec: configv1.APIServerSpec{
			TLSAdherence: configv1.TLSAdherencePolicyStrictAllComponents,
			TLSSecurityProfile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileIntermediateType,
			},
		},
	}
	r := testReconciler(t)
	r.CtrlClient = fakeCtrlClientWithAPIServer(apiServer)

	tm := testTrustManager().Build()
	dep, err := r.getDeploymentObject(tm, getResourceLabels(tm), getResourceAnnotations(tm), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected, err := tlsprofile.EffectiveSpec(apiServer.Spec.TLSSecurityProfile)
	if err != nil {
		t.Fatal(err)
	}
	want := tlsprofile.TrustManagerWebhookTLSArgs(expected)
	gotArgs := dep.Spec.Template.Spec.Containers[0].Args
	for _, w := range want {
		found := false
		for _, g := range gotArgs {
			if g == w {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected arg %q in %#v", w, gotArgs)
		}
	}
}

func fakeCtrlClientWithAPIServer(apiServer *configv1.APIServer) *fakes.FakeCtrlClient {
	mock := &fakes.FakeCtrlClient{}
	mock.GetCalls(func(_ context.Context, key client.ObjectKey, obj client.Object) error {
		if key.Name != apiServerClusterName {
			return apierrors.NewNotFound(schema.GroupResource{Group: configv1.GroupName, Resource: "apiservers"}, key.Name)
		}
		if apiServer == nil {
			return apierrors.NewNotFound(schema.GroupResource{Group: configv1.GroupName, Resource: "apiservers"}, key.Name)
		}
		dst, ok := obj.(*configv1.APIServer)
		if !ok {
			return apierrors.NewBadRequest("unexpected object type")
		}
		apiServer.DeepCopyInto(dst)
		return nil
	})
	return mock
}

func containerArgMap(dep *appsv1.Deployment) map[string]string {
	argMap := map[string]string{}
	for _, c := range dep.Spec.Template.Spec.Containers {
		if c.Name != trustManagerContainerName {
			continue
		}
		for _, a := range c.Args {
			parts := strings.SplitN(a, "=", 2)
			if len(parts) == 2 {
				argMap[parts[0]] = parts[1]
			} else {
				argMap[parts[0]] = ""
			}
		}
	}
	return argMap
}
