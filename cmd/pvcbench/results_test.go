package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPrintSummaryIncludesInputs(t *testing.T) {
	latencies := []time.Duration{time.Second, 2 * time.Second}
	inputs := SummaryInputs{
		Scenario:          "burst",
		Replicas:          2,
		PVCSize:           "100Mi",
		DeleteBatchSize:   10,
		DeleteInterval:    5 * time.Second,
		PVCPollInterval:   100 * time.Millisecond,
		KubernetesVersion: "v1.30.11",
	}

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	printSummary(3*time.Second, latencies, inputs)

	_ = w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	for _, expected := range []string{
		"Scenario: burst",
		"Replicas: 2",
		"PVC Size: 100Mi",
		"Kubernetes Version: v1.30.11",
		"PVC Poll Interval: 100ms",
		"PVC Delete Latency:",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, output)
		}
	}
}
