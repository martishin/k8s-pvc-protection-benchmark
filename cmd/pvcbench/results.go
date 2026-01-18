package main

import (
	"fmt"
	"sort"
	"time"
)

type SummaryInputs struct {
	Scenario          string
	Replicas          int32
	PVCSize           string
	DeleteBatchSize   int32
	DeleteInterval    time.Duration
	PVCPollInterval   time.Duration
	KubernetesVersion string
}

func printSummary(totalDuration time.Duration, latencies []time.Duration, inputs SummaryInputs) {
	fmt.Println("\n=== Benchmark Summary ===")
	fmt.Printf("Total Duration: %s\n", totalDuration)
	fmt.Printf("Scenario: %s\n", inputs.Scenario)
	fmt.Printf("Replicas: %d\n", inputs.Replicas)
	fmt.Printf("PVC Size: %s\n", inputs.PVCSize)
	if inputs.KubernetesVersion != "" {
		fmt.Printf("Kubernetes Version: %s\n", inputs.KubernetesVersion)
	}
	if inputs.Scenario == "staggered" {
		fmt.Printf("Delete Batch Size: %d\n", inputs.DeleteBatchSize)
		fmt.Printf("Delete Interval: %s\n", inputs.DeleteInterval)
	}
	fmt.Printf("PVC Poll Interval: %s\n", inputs.PVCPollInterval)

	if len(latencies) == 0 {
		fmt.Println("No PVC deletions recorded.")
		return
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	p50 := percentile(latencies, 50)
	p90 := percentile(latencies, 90)
	p99 := percentile(latencies, 99)
	avg := average(latencies)

	fmt.Printf("PVC Delete Latency:\n")
	fmt.Printf("  Count: %d\n", len(latencies))
	fmt.Printf("  Avg:   %s\n", avg)
	fmt.Printf("  p50:   %s\n", p50)
	fmt.Printf("  p90:   %s\n", p90)
	fmt.Printf("  p99:   %s\n", p99)
	fmt.Println("==========================")
}

func percentile(latencies []time.Duration, p int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	idx := (p * len(latencies)) / 100
	if idx >= len(latencies) {
		idx = len(latencies) - 1
	}
	return latencies[idx]
}

func average(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	var total time.Duration
	for _, l := range latencies {
		total += l
	}
	return total / time.Duration(len(latencies))
}
