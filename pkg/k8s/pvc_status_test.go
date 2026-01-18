package k8s

import (
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"pvc-protection-bench/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestListPVCNames(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx := context.Background()
	namespace := "test-ns"

	for _, name := range []string{"pvc-a", "pvc-b"} {
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					"app": "pvcbench-sts",
				},
			},
		}
		_, err := client.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("create pvc: %v", err)
		}
	}

	names, err := ListPVCNames(ctx, client, namespace, "app=pvcbench-sts")
	if err != nil {
		t.Fatalf("ListPVCNames error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 pvc names, got %d", len(names))
	}
}

func TestPollPVCDeletion(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	namespace := "test-ns"
	pvcNames := []string{"pvc-1", "pvc-2"}
	callCounts := map[string]int{}

	client.PrependReactor("get", "persistentvolumeclaims", func(action k8stesting.Action) (bool, runtime.Object, error) {
		get := action.(k8stesting.GetAction)
		name := get.GetName()
		callCounts[name]++

		if callCounts[name] == 1 {
			now := metav1.NewTime(time.Now())
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:              name,
					Namespace:         namespace,
					DeletionTimestamp: &now,
				},
			}
			return true, pvc, nil
		}

		return true, nil, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "persistentvolumeclaims"}, name)
	})

	latencies, err := PollPVCDeletion(ctx, client, namespace, pvcNames, "burst", "100Mi", 2, "single", 1*time.Millisecond)
	if err != nil {
		t.Fatalf("PollPVCDeletion error: %v", err)
	}
	if len(latencies) != len(pvcNames) {
		t.Fatalf("expected %d latencies, got %d", len(pvcNames), len(latencies))
	}
}

func TestPollPVCDeletionResetsTerminatingOnError(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	namespace := "test-ns"
	pvcNames := []string{"pvc-1"}

	client.PrependReactor("get", "persistentvolumeclaims", func(action k8stesting.Action) (bool, runtime.Object, error) {
		now := metav1.NewTime(time.Now())
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pvc-1",
				Namespace:         namespace,
				DeletionTimestamp: &now,
			},
		}
		return true, pvc, apierrors.NewInternalError(fmt.Errorf("boom"))
	})

	_, err := PollPVCDeletion(ctx, client, namespace, pvcNames, "burst", "100Mi", 1, "single", 1*time.Millisecond)
	if err == nil {
		t.Fatalf("expected error")
	}

	if val := testutil.ToFloat64(metrics.PVCsTerminating); val != 0 {
		t.Fatalf("expected PVCsTerminating to be reset to 0, got %v", val)
	}
}
