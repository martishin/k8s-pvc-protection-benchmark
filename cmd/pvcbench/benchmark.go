package main

import (
	"context"
	"fmt"
	"time"

	"pvc-protection-bench/pkg/k8s"
	"pvc-protection-bench/pkg/metrics"
	"pvc-protection-bench/pkg/scenarios"

	"github.com/spf13/cobra"
)

var (
	scenario        string
	replicas        int32
	pvcSize         string
	batchSize       int32
	deleteInterval  time.Duration
	pvcPollInterval time.Duration
)

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run a single benchmark scenario",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateBenchmarkInputs(scenario, replicas, pvcSize, batchSize, deleteInterval, pvcPollInterval); err != nil {
			return err
		}

		client, err := k8s.NewClient(clientQPS, clientBurst)
		if err != nil {
			return err
		}

		k8sVersion := "unknown"
		serverVersion, err := client.Discovery().ServerVersion()
		if err == nil {
			k8sVersion = serverVersion.GitVersion
		}

		namespace := fmt.Sprintf("pvcbench-%d", time.Now().Unix())
		metrics.StartMetricsServer(metricsPort, namespace)

		config := k8s.StatefulSetConfig{
			Name:      "pvcbench-sts",
			Namespace: namespace,
			Replicas:  replicas,
			PVCSize:   pvcSize,
		}

		ctx := context.Background()

		var totalDuration time.Duration
		var latencies []time.Duration

		switch scenario {
		case "burst":
			totalDuration, latencies, err = scenarios.RunBurstDelete(ctx, client, config, pvcPollInterval)
		case "staggered":
			opts := scenarios.StaggeredDeleteOptions{
				BatchSize: batchSize,
				Interval:  deleteInterval,
			}
			totalDuration, latencies, err = scenarios.RunStaggeredDelete(ctx, client, config, opts, pvcPollInterval)
		default:
			return fmt.Errorf("unknown scenario: %s", scenario)
		}

		if err == nil {
			summaryInputs := SummaryInputs{
				Scenario:          scenario,
				Replicas:          replicas,
				PVCSize:           pvcSize,
				DeleteBatchSize:   batchSize,
				DeleteInterval:    deleteInterval,
				PVCPollInterval:   pvcPollInterval,
				KubernetesVersion: k8sVersion,
			}
			printSummary(totalDuration, latencies, summaryInputs)
		}

		return err
	},
}

func init() {
	benchmarkCmd.Flags().StringVar(&scenario, "scenario", "burst", "Scenario to run: burst (all-at-once), staggered (batched)")
	benchmarkCmd.Flags().Int32Var(&replicas, "replicas", 100, "Number of replicas")
	benchmarkCmd.Flags().StringVar(&pvcSize, "pvc-size", "100Mi", "PVC size")

	benchmarkCmd.Flags().Int32Var(&batchSize, "delete-batch-size", 10, "Batch size for staggered scenario")
	benchmarkCmd.Flags().DurationVar(&deleteInterval, "delete-interval", 5*time.Second, "Interval between batches for staggered scenario")
	benchmarkCmd.Flags().DurationVar(&pvcPollInterval, "pvc-poll-interval", 100*time.Millisecond, "Interval for PVC GET polling")

	rootCmd.AddCommand(benchmarkCmd)
}

func validateBenchmarkInputs(scenario string, replicas int32, pvcSize string, batchSize int32, deleteInterval, pvcPollInterval time.Duration) error {
	if scenario != "burst" && scenario != "staggered" {
		return fmt.Errorf("unknown scenario: %s", scenario)
	}
	if replicas <= 0 {
		return fmt.Errorf("replicas must be > 0 (got %d)", replicas)
	}
	if pvcSize == "" {
		return fmt.Errorf("pvc-size must be set")
	}
	if pvcPollInterval <= 0 {
		return fmt.Errorf("pvc-poll-interval must be > 0 (got %s)", pvcPollInterval)
	}
	if scenario == "staggered" {
		if batchSize <= 0 || batchSize > replicas {
			return fmt.Errorf("delete-batch-size must be > 0 and <= replicas (got %d, replicas=%d)", batchSize, replicas)
		}
		if deleteInterval <= 0 {
			return fmt.Errorf("delete-interval must be > 0 (got %s)", deleteInterval)
		}
	}
	return nil
}
