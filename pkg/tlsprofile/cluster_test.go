package tlsprofile

import (
	"context"
	"errors"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	configv1 "github.com/openshift/api/config/v1"
)

func TestResolveHonoredTLSProfile_fetchErrors(t *testing.T) {
	notFound := apierrors.NewNotFound(schema.GroupResource{Group: configv1.GroupName, Resource: "apiservers"}, APIServerClusterName)
	forbidden := apierrors.NewForbidden(schema.GroupResource{Group: configv1.GroupName, Resource: "apiservers"}, APIServerClusterName, errors.New("denied"))

	t.Run("NotFound is soft skip", func(t *testing.T) {
		spec, err := ResolveHonoredTLSProfile(context.Background(), func(context.Context) (*configv1.APIServer, error) {
			return nil, notFound
		}, "test", FetchErrorPropagateExceptNotFound)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if spec != nil {
			t.Fatalf("expected nil spec on NotFound, got %#v", spec)
		}
	})

	t.Run("Forbidden propagates", func(t *testing.T) {
		spec, err := ResolveHonoredTLSProfile(context.Background(), func(context.Context) (*configv1.APIServer, error) {
			return nil, forbidden
		}, "test", FetchErrorPropagateExceptNotFound)
		if err == nil {
			t.Fatal("expected error")
		}
		if spec != nil {
			t.Fatalf("expected nil spec on Forbidden, got %#v", spec)
		}
	})
}
