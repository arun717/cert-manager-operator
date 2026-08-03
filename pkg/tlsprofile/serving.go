package tlsprofile

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	configv1 "github.com/openshift/api/config/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned"
	libgocrypto "github.com/openshift/library-go/pkg/crypto"
)

const apiServerClusterName = "cluster"

// ApplyClusterProfileToHTTPServingInfo reads apiserver.config.openshift.io/cluster
// and, when tlsAdherence requires enforcement, applies the effective TLS profile
// to serving. Missing APIServer (non-OpenShift) or non-enforcing adherence leaves
// serving unchanged.
func ApplyClusterProfileToHTTPServingInfo(ctx context.Context, restConfig *rest.Config, serving *configv1.HTTPServingInfo) error {
	if restConfig == nil {
		return fmt.Errorf("rest config is nil")
	}
	if serving == nil {
		return fmt.Errorf("HTTPServingInfo is nil")
	}

	configClient, err := configv1client.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create config client: %w", err)
	}

	apiServer, err := configClient.ConfigV1().APIServers().Get(ctx, apiServerClusterName, metav1.GetOptions{})
	if err != nil {
		// Non-OpenShift or RBAC/API unavailable: keep library-go defaults.
		klog.V(2).Infof("skipping cluster TLS profile for operator serving: failed to get apiserver/cluster: %v", err)
		return nil
	}

	adherence := apiServer.Spec.TLSAdherence
	if !libgocrypto.ShouldHonorClusterTLSProfile(adherence) {
		klog.V(2).Infof("skipping cluster TLS profile for operator serving: apiserver tlsAdherence=%q", adherence)
		return nil
	}
	if adherence != configv1.TLSAdherencePolicyStrictAllComponents {
		klog.Warningf("apiserver.config.openshift.io/cluster has unknown tlsAdherence %q; treating as StrictAllComponents for operator serving", adherence)
	}

	effective, err := EffectiveSpec(apiServer.Spec.TLSSecurityProfile)
	if err != nil {
		return err
	}
	if err := ApplyToHTTPServingInfo(serving, effective); err != nil {
		return err
	}
	klog.V(2).Infof("applied cluster TLS profile to operator serving: minTLSVersion=%s ciphers=%d", serving.MinTLSVersion, len(serving.CipherSuites))
	return nil
}

// RESTConfigFromKubeConfig returns an in-cluster rest.Config when kubeConfigFile
// is empty, otherwise loads the given kubeconfig path.
func RESTConfigFromKubeConfig(kubeConfigFile string) (*rest.Config, error) {
	if len(kubeConfigFile) == 0 {
		return rest.InClusterConfig()
	}
	return clientcmd.BuildConfigFromFlags("", kubeConfigFile)
}
