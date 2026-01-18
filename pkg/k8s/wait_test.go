package k8s

import (
	"context"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestWaitForStatefulSetDeleted(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client.PrependReactor("get", "statefulsets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewNotFound(schema.GroupResource{Group: "apps", Resource: "statefulsets"}, "pvcbench-sts")
	})

	if err := WaitForStatefulSetDeleted(ctx, client, "pvcbench-ns", "pvcbench-sts"); err != nil {
		t.Fatalf("WaitForStatefulSetDeleted error: %v", err)
	}
}
