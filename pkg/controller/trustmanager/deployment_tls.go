package trustmanager

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	configv1 "github.com/openshift/api/config/v1"
	libgocrypto "github.com/openshift/library-go/pkg/crypto"

	"github.com/openshift/cert-manager-operator/pkg/controller/common"
	"github.com/openshift/cert-manager-operator/pkg/tlsprofile"
)

const apiServerClusterName = "cluster"

// applyClusterTLSProfile merges cluster TLS security profile flags onto the
// trust-manager webhook container when apiserver tlsAdherence requires it.
// When the APIServer resource is missing (non-OpenShift) or adherence does not
// require enforcement, this is a no-op.
func (r *Reconciler) applyClusterTLSProfile(deployment *appsv1.Deployment) error {
	if r.CtrlClient == nil {
		return nil
	}

	apiServer := &configv1.APIServer{}
	if err := r.Get(r.ctx, types.NamespacedName{Name: apiServerClusterName}, apiServer); err != nil {
		if apierrors.IsNotFound(err) {
			klog.V(4).Info("skipping cluster TLS profile for trust-manager: apiserver.config.openshift.io/cluster not found")
			return nil
		}
		return fmt.Errorf("failed to get apiserver.config.openshift.io/cluster: %w", err)
	}

	adherence := apiServer.Spec.TLSAdherence
	if !libgocrypto.ShouldHonorClusterTLSProfile(adherence) {
		klog.V(4).Infof("skipping cluster TLS profile for trust-manager: apiserver tlsAdherence=%q", adherence)
		return nil
	}
	if adherence != configv1.TLSAdherencePolicyStrictAllComponents {
		klog.Warningf("apiserver.config.openshift.io/cluster has unknown tlsAdherence %q; treating as StrictAllComponents for trust-manager", adherence)
	}

	effective, err := tlsprofile.EffectiveSpec(apiServer.Spec.TLSSecurityProfile)
	if err != nil {
		return err
	}

	return applyTrustManagerWebhookTLSArgs(deployment, effective)
}

// applyTrustManagerWebhookTLSArgs merges profile-derived webhook TLS flags onto
// the trust-manager container. Exported for unit tests via package-level use.
func applyTrustManagerWebhookTLSArgs(deployment *appsv1.Deployment, spec *configv1.TLSProfileSpec) error {
	extra := tlsprofile.TrustManagerWebhookTLSArgs(spec)
	if len(extra) == 0 {
		return nil
	}

	for i := range deployment.Spec.Template.Spec.Containers {
		if deployment.Spec.Template.Spec.Containers[i].Name != trustManagerContainerName {
			continue
		}
		sourceArgs := deployment.Spec.Template.Spec.Containers[i].Args
		if spec != nil && spec.MinTLSVersion == configv1.VersionTLS13 {
			sourceArgs = common.StripArgsByKeys(sourceArgs, common.ArgKeysSet(tlsprofile.TrustManagerCipherSuiteArgKeys))
		}
		deployment.Spec.Template.Spec.Containers[i].Args = common.MergeContainerArgs(sourceArgs, extra)
		return nil
	}
	return fmt.Errorf("deployment %s/%s missing container %q", deployment.Namespace, deployment.Name, trustManagerContainerName)
}
