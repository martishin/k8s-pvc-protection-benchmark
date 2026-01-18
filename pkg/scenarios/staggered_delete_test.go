package scenarios

import (
	"context"
	"fmt"
	"testing"
	"time"

	"pvc-protection-bench/pkg/k8s"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestRunStaggeredDelete(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	config := k8s.StatefulSetConfig{
		Name:      "pvcbench-sts",
		Namespace: "pvcbench-test",
		Replicas:  2,
		PVCSize:   "100Mi",
	}
	opts := StaggeredDeleteOptions{
		BatchSize: 1,
		Interval:  0,
	}

	client.PrependReactor("get", "statefulsets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		get := action.(k8stesting.GetAction)
		obj, err := client.Tracker().Get(appsv1.SchemeGroupVersion.WithResource("statefulsets"), get.GetNamespace(), get.GetName())
		if err != nil {
			return true, nil, err
		}
		sts := obj.(*appsv1.StatefulSet).DeepCopy()
		if sts.Spec.Replicas != nil {
			sts.Status.ReadyReplicas = *sts.Spec.Replicas
		}
		return true, sts, nil
	})

	if _, err := client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: config.Namespace},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	for i := 0; i < int(config.Replicas); i++ {
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pvc-%d", i),
				Namespace: config.Namespace,
				Labels: map[string]string{
					"app": config.Name,
				},
			},
		}
		_, err := client.CoreV1().PersistentVolumeClaims(config.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("create pvc: %v", err)
		}
	}

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
					Namespace:         config.Namespace,
					DeletionTimestamp: &now,
				},
			}
			return true, pvc, nil
		}
		return true, nil, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "persistentvolumeclaims"}, name)
	})

	_, latencies, err := RunStaggeredDelete(ctx, client, config, opts, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("RunStaggeredDelete error: %v", err)
	}
	if len(latencies) != int(config.Replicas) {
		t.Fatalf("expected %d latencies, got %d", config.Replicas, len(latencies))
	}
}
