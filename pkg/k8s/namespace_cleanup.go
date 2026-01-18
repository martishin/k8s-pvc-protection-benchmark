package k8s

import (
	"context"
	"time"

	"pvc-protection-bench/pkg/logging"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func WaitForNamespaceDeleted(ctx context.Context, client kubernetes.Interface, name string) error {
	return wait.PollImmediate(1*time.Second, 10*time.Minute, func() (bool, error) {
		_, err := client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			return false, nil
		}
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

func ForceDeleteNamespace(ctx context.Context, client kubernetes.Interface, name string) error {
	logger := logging.GetLogger()

	pvcs, err := client.CoreV1().PersistentVolumeClaims(name).List(ctx, metav1.ListOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	for _, pvc := range pvcs.Items {
		if len(pvc.Finalizers) == 0 {
			continue
		}
		patch := []byte(`{"metadata":{"finalizers":[]}}`)
		_, err := client.CoreV1().PersistentVolumeClaims(name).Patch(ctx, pvc.Name, types.MergePatchType, patch, metav1.PatchOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			logger.Error("failed to remove pvc finalizers", logging.StringField("pvc", pvc.Name), logging.ErrorField(err))
			return err
		}
	}

	nsPatch := []byte(`{"spec":{"finalizers":[]}}`)
	_, err = client.CoreV1().Namespaces().Patch(ctx, name, types.MergePatchType, nsPatch, metav1.PatchOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return WaitForNamespaceDeleted(ctx, client, name)
}
