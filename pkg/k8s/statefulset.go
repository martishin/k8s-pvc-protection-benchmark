package k8s

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

type StatefulSetConfig struct {
	Name      string
	Namespace string
	Replicas  int32
	PVCSize   string
}

func CreateStatefulSet(ctx context.Context, client kubernetes.Interface, config StatefulSetConfig) (*appsv1.StatefulSet, error) {
	deletePolicy := appsv1.DeletePersistentVolumeClaimRetentionPolicyType
	labels := map[string]string{
		"app": config.Name,
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &config.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": config.Name,
				},
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "pause",
							Image: "registry.k8s.io/pause:3.9",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/mnt/data",
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "data",
						Labels: labels,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse(config.PVCSize),
							},
						},
					},
				},
			},
			PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
				WhenScaled:  deletePolicy,
				WhenDeleted: deletePolicy,
			},
		},
	}

	return client.AppsV1().StatefulSets(config.Namespace).Create(ctx, sts, metav1.CreateOptions{})
}

func ScaleStatefulSet(ctx context.Context, client kubernetes.Interface, namespace, name string, replicas int32) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		sts, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		sts.Spec.Replicas = &replicas
		_, err = client.AppsV1().StatefulSets(namespace).Update(ctx, sts, metav1.UpdateOptions{})
		return err
	})
}

func DeleteStatefulSet(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	return client.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
