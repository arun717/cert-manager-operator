package certmanager

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWithOperandMetricsTLS(t *testing.T) {
	tests := []struct {
		name           string
		deploymentName string
		wantArgs       []string
		wantScheme     string
	}{
		{
			name:           "controller",
			deploymentName: certmanagerControllerDeployment,
			wantArgs: []string{
				"--metrics-dynamic-serving-ca-secret-namespace=$(POD_NAMESPACE)",
				"--metrics-dynamic-serving-ca-secret-name=cert-manager-metrics-ca",
				"--metrics-dynamic-serving-dns-names=cert-manager,cert-manager.$(POD_NAMESPACE),cert-manager.$(POD_NAMESPACE).svc",
			},
			wantScheme: "https",
		},
		{
			name:           "webhook",
			deploymentName: certmanagerWebhookDeployment,
			wantArgs: []string{
				"--metrics-dynamic-serving-ca-secret-name=cert-manager-metrics-ca",
				"--metrics-dynamic-serving-dns-names=cert-manager-webhook,cert-manager-webhook.$(POD_NAMESPACE),cert-manager-webhook.$(POD_NAMESPACE).svc",
			},
			wantScheme: "https",
		},
		{
			name:           "cainjector",
			deploymentName: certmanagerCAinjectorDeployment,
			wantArgs: []string{
				"--metrics-dynamic-serving-ca-secret-name=cert-manager-metrics-ca",
				"--metrics-dynamic-serving-dns-names=cert-manager-cainjector,cert-manager-cainjector.$(POD_NAMESPACE),cert-manager-cainjector.$(POD_NAMESPACE).svc",
			},
			wantScheme: "https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: tt.deploymentName, Namespace: "cert-manager"},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"prometheus.io/port": "9402",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name: tt.deploymentName,
								Args: []string{"--v=2"},
							}},
						},
					},
				},
			}
			if err := withOperandMetricsTLS(nil, dep); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			argMap := map[string]string{}
			for _, a := range dep.Spec.Template.Spec.Containers[0].Args {
				parts := strings.SplitN(a, "=", 2)
				if len(parts) == 2 {
					argMap[parts[0]] = parts[1]
				}
			}
			for _, want := range tt.wantArgs {
				parts := strings.SplitN(want, "=", 2)
				if argMap[parts[0]] != parts[1] {
					t.Fatalf("arg %s: got %q want %q (all=%#v)", parts[0], argMap[parts[0]], parts[1], argMap)
				}
			}
			if dep.Spec.Template.Annotations["prometheus.io/scheme"] != tt.wantScheme {
				t.Fatalf("scheme annotation: %q", dep.Spec.Template.Annotations["prometheus.io/scheme"])
			}
		})
	}
}
