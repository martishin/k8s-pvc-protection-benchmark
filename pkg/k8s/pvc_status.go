package k8s

import (
	"context"
	"fmt"
	"time"

	"pvc-protection-bench/pkg/metrics"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ListPVCNames(ctx context.Context, client kubernetes.Interface, namespace, labelSelector string) ([]string, error) {
	pvcs, err := client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(pvcs.Items))
	for _, pvc := range pvcs.Items {
		names = append(names, pvc.Name)
	}
	return names, nil
}

func PollPVCDeletion(ctx context.Context, client kubernetes.Interface, namespace string, pvcNames []string, scenario, pvcSize string, replicas int, nsGroup string, pollInterval time.Duration) (_ []time.Duration, errRet error) {
	if len(pvcNames) == 0 {
		return nil, fmt.Errorf("no PVCs found to track deletion")
	}

	startTimes := make(map[string]time.Time, len(pvcNames))
	terminating := make(map[string]bool, len(pvcNames))
	done := make(map[string]bool, len(pvcNames))
	latencies := make([]time.Duration, 0, len(pvcNames))

	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	defer func() {
		if errRet != nil {
			metrics.PVCsTerminating.Set(0)
		}
	}()

	for {
		allDone := true
		for _, name := range pvcNames {
			if done[name] {
				continue
			}
			allDone = false

			pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					start, ok := startTimes[name]
					if !ok {
						start = time.Now()
					}
					latency := time.Since(start)
					latencies = append(latencies, latency)
					metrics.PVCDeleteLatency.WithLabelValues(scenario, pvcSize, fmt.Sprintf("%d", replicas), nsGroup).Observe(latency.Seconds())
					if terminating[name] {
						metrics.PVCsTerminating.Dec()
					}
					done[name] = true
					continue
				}
				errRet = err
				return nil, err
			}

			if pvc.DeletionTimestamp != nil {
				if _, ok := startTimes[name]; !ok {
					start := pvc.DeletionTimestamp.Time
					if start.IsZero() {
						start = time.Now()
					}
					startTimes[name] = start
					terminating[name] = true
					metrics.PVCsTerminating.Inc()
				}
			}
		}

		if allDone {
			break
		}

		select {
		case <-ctx.Done():
			errRet = ctx.Err()
			return nil, errRet
		case <-ticker.C:
		}
	}

	return latencies, nil
}
