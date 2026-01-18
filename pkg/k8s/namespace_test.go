package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEnsureNamespaceCreatesWhenMissing(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx := context.Background()
	name := "pvcbench-test"

	if err := EnsureNamespace(ctx, client, name); err != nil {
		t.Fatalf("EnsureNamespace error: %v", err)
	}

	ns, err := client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected namespace to exist: %v", err)
	}
	if ns.Name != name {
		t.Fatalf("expected namespace name %s, got %s", name, ns.Name)
	}
}

func TestDeleteNamespaceIgnoresNotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx := context.Background()

	if err := DeleteNamespace(ctx, client, "missing"); err != nil {
		t.Fatalf("DeleteNamespace should ignore NotFound: %v", err)
	}
}

func TestEnsureNamespaceNoopWhenExists(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx := context.Background()

	_, err := client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "pvcbench-test"},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	if err := EnsureNamespace(ctx, client, "pvcbench-test"); err != nil {
		t.Fatalf("EnsureNamespace error: %v", err)
	}
}
