package k8s

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func WaitForStatefulSetReady(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	return wait.PollImmediate(1*time.Second, 10*time.Minute, func() (bool, error) {
		sts, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if sts.Status.ReadyReplicas == *sts.Spec.Replicas {
			return true, nil
		}
		return false, nil
	})
}

func WaitForStatefulSetDeleted(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	return wait.PollImmediate(1*time.Second, 10*time.Minute, func() (bool, error) {
		_, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			return false, nil
		}
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}
