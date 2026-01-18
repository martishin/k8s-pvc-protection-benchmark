package k8s

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestForceDeleteNamespacePatchesFinalizers(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "pvcbench-test",
			Finalizers: []string{"kubernetes"},
		},
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "pvc-1",
			Namespace:  ns.Name,
			Finalizers: []string{"kubernetes.io/pvc-protection"},
		},
	}

	_, err := client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	_, err = client.CoreV1().PersistentVolumeClaims(ns.Name).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("create pvc: %v", err)
	}

	var pvcPatchCount int32
	var nsPatchCount int32

	client.PrependReactor("patch", "persistentvolumeclaims", func(action k8stesting.Action) (bool, runtime.Object, error) {
		atomic.AddInt32(&pvcPatchCount, 1)
		return true, pvc, nil
	})
	client.PrependReactor("patch", "namespaces", func(action k8stesting.Action) (bool, runtime.Object, error) {
		atomic.AddInt32(&nsPatchCount, 1)
		return true, ns, nil
	})
	client.PrependReactor("get", "namespaces", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "namespaces"}, ns.Name)
	})

	if err := ForceDeleteNamespace(ctx, client, ns.Name); err != nil {
		t.Fatalf("ForceDeleteNamespace error: %v", err)
	}
	if pvcPatchCount == 0 {
		t.Fatalf("expected pvc patch to be called")
	}
	if nsPatchCount == 0 {
		t.Fatalf("expected namespace patch to be called")
	}
}

func TestWaitForNamespaceDeleted(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	name := "pvcbench-test"

	client.PrependReactor("get", "namespaces", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "namespaces"}, name)
	})

	if err := WaitForNamespaceDeleted(ctx, client, name); err != nil {
		t.Fatalf("WaitForNamespaceDeleted error: %v", err)
	}
}
