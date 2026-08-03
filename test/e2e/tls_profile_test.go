//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"

	configapiv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/cert-manager-operator/pkg/tlsprofile"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cluster TLS security profile", Label("Platform:Generic", "Feature:TLSProfile", "TechPreview"), Ordered, func() {
	var ctx context.Context

	BeforeAll(func() {
		ctx = context.Background()
	})

	BeforeEach(func() {
		By("waiting for operator status to become available")
		err := VerifyHealthyOperatorConditions(certmanageroperatorclient.OperatorV1alpha1())
		Expect(err).NotTo(HaveOccurred(), "Operator is expected to be available")
	})

	It("should configure operand container TLS args from apiserver cluster profile", func() {
		original, err := getClusterAPIServerTLSConfig(ctx)
		if apierrors.IsNotFound(err) {
			Skip("apiserver.config.openshift.io/cluster is not available on this cluster")
		}
		Expect(err).NotTo(HaveOccurred(), "failed to read apiserver TLS configuration")

		testProfile := &configapiv1.TLSSecurityProfile{
			Type: configapiv1.TLSProfileModernType,
		}
		strictAdherence := configapiv1.TLSAdherencePolicyStrictAllComponents

		DeferCleanup(func() {
			By("[cleanup] restoring original apiserver TLS configuration")
			Eventually(func() error {
				return restoreClusterAPIServerTLSConfig(ctx, original)
			}, lowTimeout, fastPollInterval).Should(Succeed())
		})

		By("patching apiserver cluster to enforce StrictAllComponents with Modern TLS profile")
		err = updateClusterAPIServerTLSConfig(ctx, testProfile, strictAdherence)
		if isTLSAdherenceUnsupported(err) {
			Skip(fmt.Sprintf("apiserver tlsAdherence is not available on this cluster: %v", err))
		}
		Expect(err).NotTo(HaveOccurred(), "failed to patch apiserver TLS configuration")

		expectedSpec, err := tlsprofile.EffectiveSpec(testProfile)
		Expect(err).NotTo(HaveOccurred(), "failed to resolve expected TLS profile spec")

		By("verifying cert-manager operand deployments expose cluster TLS flags")
		for _, name := range []string{
			certmanagerControllerDeployment,
			certmanagerWebhookDeployment,
			certmanagerCAinjectorDeployment,
		} {
			err := verifyOperandTLSArgsMatchClusterProfile(name, expectedSpec)
			Expect(err).NotTo(HaveOccurred(), "deployment %s", name)
		}

		By("verifying trust-manager webhook TLS flags when the deployment is present")
		_, tmErr := k8sClientSet.AppsV1().Deployments(operandNamespace).Get(ctx, "trust-manager", metav1.GetOptions{})
		if apierrors.IsNotFound(tmErr) {
			Skip("trust-manager deployment not present; skipping trust-manager TLS profile verification")
		}
		Expect(tmErr).NotTo(HaveOccurred(), "failed to get trust-manager deployment")
		err = verifyOperandTLSArgsMatchClusterProfile("trust-manager", expectedSpec)
		Expect(err).NotTo(HaveOccurred(), "deployment trust-manager")
	})
})
