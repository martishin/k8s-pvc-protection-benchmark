package k8s

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateStatefulSetSpec(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx := context.Background()

	config := StatefulSetConfig{
		Name:      "pvcbench-sts",
		Namespace: "pvcbench-1",
		Replicas:  5,
		PVCSize:   "100Mi",
	}

	sts, err := CreateStatefulSet(ctx, client, config)
	if err != nil {
		t.Fatalf("CreateStatefulSet error: %v", err)
	}

	if sts.Spec.PodManagementPolicy != appsv1.ParallelPodManagement {
		t.Fatalf("expected PodManagementPolicy=Parallel, got %q", sts.Spec.PodManagementPolicy)
	}
	if sts.Spec.PersistentVolumeClaimRetentionPolicy == nil {
		t.Fatalf("expected PVC retention policy to be set")
	}
	if sts.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled != appsv1.DeletePersistentVolumeClaimRetentionPolicyType {
		t.Fatalf("expected WhenScaled=Delete")
	}
	if sts.Spec.PersistentVolumeClaimRetentionPolicy.WhenDeleted != appsv1.DeletePersistentVolumeClaimRetentionPolicyType {
		t.Fatalf("expected WhenDeleted=Delete")
	}
	if len(sts.Spec.VolumeClaimTemplates) != 1 {
		t.Fatalf("expected 1 volumeClaimTemplate, got %d", len(sts.Spec.VolumeClaimTemplates))
	}

	pvc := sts.Spec.VolumeClaimTemplates[0]
	if pvc.ObjectMeta.Labels["app"] != config.Name {
		t.Fatalf("expected pvc label app=%s", config.Name)
	}
	req := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if req.String() != config.PVCSize {
		t.Fatalf("expected pvc size %s, got %s", config.PVCSize, req.String())
	}
}
