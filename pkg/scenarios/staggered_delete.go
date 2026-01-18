package scenarios

import (
	"context"
	"fmt"
	"time"

	"pvc-protection-bench/pkg/k8s"
	"pvc-protection-bench/pkg/logging"
	"pvc-protection-bench/pkg/metrics"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type StaggeredDeleteOptions struct {
	BatchSize int32
	Interval  time.Duration
}

func RunStaggeredDelete(ctx context.Context, client kubernetes.Interface, config k8s.StatefulSetConfig, opts StaggeredDeleteOptions, pollInterval time.Duration) (time.Duration, []time.Duration, error) {
	replicaStr := fmt.Sprintf("%d", config.Replicas)
	metrics.RunInfo.WithLabelValues("staggered", config.PVCSize, replicaStr).Set(1)
	defer metrics.RunInfo.WithLabelValues("staggered", config.PVCSize, replicaStr).Set(0)

	logger := logging.GetLogger().With(
		logging.StringField("scenario", "staggered"),
		logging.StringField("namespace", config.Namespace),
	)

	logger.Info("starting staggered scenario")

	if err := k8s.EnsureNamespace(ctx, client, config.Namespace); err != nil {
		metrics.ErrorsTotal.WithLabelValues("namespace_creation").Inc()
		return 0, nil, err
	}

	existing, err := client.AppsV1().StatefulSets(config.Namespace).Get(ctx, config.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		metrics.ErrorsTotal.WithLabelValues("sts_get").Inc()
		return 0, nil, err
	}
	if err == nil && existing != nil {
		if err := k8s.DeleteStatefulSet(ctx, client, config.Namespace, config.Name); err != nil {
			metrics.ErrorsTotal.WithLabelValues("sts_delete").Inc()
			return 0, nil, err
		}
		if err := k8s.WaitForStatefulSetDeleted(ctx, client, config.Namespace, config.Name); err != nil {
			metrics.ErrorsTotal.WithLabelValues("sts_delete_wait").Inc()
			return 0, nil, err
		}
	}

	sts, err := k8s.CreateStatefulSet(ctx, client, config)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("sts_creation").Inc()
		return 0, nil, err
	}

	logger.Info("waiting for pods to be ready")
	if err := k8s.WaitForStatefulSetReady(ctx, client, config.Namespace, sts.Name); err != nil {
		metrics.ErrorsTotal.WithLabelValues("sts_ready_wait").Inc()
		return 0, nil, err
	}

	labelSelector := fmt.Sprintf("app=%s", sts.Name)
	pvcNames, err := k8s.ListPVCNames(ctx, client, config.Namespace, labelSelector)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("pvc_list").Inc()
		return 0, nil, err
	}

	logger.Info("scaling down in batches")
	metrics.PodsRemaining.Set(float64(config.Replicas))
	start := time.Now()

	currentReplicas := config.Replicas
	for currentReplicas > 0 {
		currentReplicas -= opts.BatchSize
		if currentReplicas < 0 {
			currentReplicas = 0
		}

		logger.Info("scaling down", logging.StringField("replicas", fmt.Sprintf("%d", currentReplicas)))
		if err := k8s.ScaleStatefulSet(ctx, client, config.Namespace, sts.Name, currentReplicas); err != nil {
			metrics.ErrorsTotal.WithLabelValues("sts_scale").Inc()
			return 0, nil, err
		}
		metrics.PodsRemaining.Set(float64(currentReplicas))

		if currentReplicas > 0 {
			time.Sleep(opts.Interval)
		}
	}

	latencies, err := k8s.PollPVCDeletion(ctx, client, config.Namespace, pvcNames, "staggered", config.PVCSize, int(config.Replicas), "single", pollInterval)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("pvc_delete_poll").Inc()
		return 0, nil, err
	}

	totalDuration := time.Since(start)
	metrics.PodsRemaining.Set(0)
	logger.Info("staggered completed", logging.StringField("duration", totalDuration.String()))

	metrics.TotalDuration.WithLabelValues("staggered", config.PVCSize, replicaStr).Set(totalDuration.Seconds())

	return totalDuration, latencies, nil
}
