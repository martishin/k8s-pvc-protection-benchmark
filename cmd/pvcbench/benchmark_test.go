package main

import (
	"testing"
	"time"
)

func TestValidateBenchmarkInputs(t *testing.T) {
	tests := []struct {
		name           string
		scenario       string
		replicas       int32
		pvcSize        string
		batchSize      int32
		deleteInterval time.Duration
		pollInterval   time.Duration
		wantErr        bool
	}{
		{
			name:           "burst-valid",
			scenario:       "burst",
			replicas:       10,
			pvcSize:        "100Mi",
			batchSize:      1,
			deleteInterval: 1 * time.Second,
			pollInterval:   100 * time.Millisecond,
			wantErr:        false,
		},
		{
			name:           "staggered-valid",
			scenario:       "staggered",
			replicas:       10,
			pvcSize:        "100Mi",
			batchSize:      5,
			deleteInterval: 1 * time.Second,
			pollInterval:   100 * time.Millisecond,
			wantErr:        false,
		},
		{
			name:           "bad-scenario",
			scenario:       "burst-delete",
			replicas:       10,
			pvcSize:        "100Mi",
			batchSize:      1,
			deleteInterval: 1 * time.Second,
			pollInterval:   100 * time.Millisecond,
			wantErr:        true,
		},
		{
			name:           "replicas-zero",
			scenario:       "burst",
			replicas:       0,
			pvcSize:        "100Mi",
			batchSize:      1,
			deleteInterval: 1 * time.Second,
			pollInterval:   100 * time.Millisecond,
			wantErr:        true,
		},
		{
			name:           "pvc-size-empty",
			scenario:       "burst",
			replicas:       10,
			pvcSize:        "",
			batchSize:      1,
			deleteInterval: 1 * time.Second,
			pollInterval:   100 * time.Millisecond,
			wantErr:        true,
		},
		{
			name:           "poll-interval-zero",
			scenario:       "burst",
			replicas:       10,
			pvcSize:        "100Mi",
			batchSize:      1,
			deleteInterval: 1 * time.Second,
			pollInterval:   0,
			wantErr:        true,
		},
		{
			name:           "staggered-batch-too-large",
			scenario:       "staggered",
			replicas:       10,
			pvcSize:        "100Mi",
			batchSize:      11,
			deleteInterval: 1 * time.Second,
			pollInterval:   100 * time.Millisecond,
			wantErr:        true,
		},
		{
			name:           "staggered-interval-zero",
			scenario:       "staggered",
			replicas:       10,
			pvcSize:        "100Mi",
			batchSize:      5,
			deleteInterval: 0,
			pollInterval:   100 * time.Millisecond,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		err := validateBenchmarkInputs(tt.scenario, tt.replicas, tt.pvcSize, tt.batchSize, tt.deleteInterval, tt.pollInterval)
		if tt.wantErr && err == nil {
			t.Fatalf("%s: expected error, got nil", tt.name)
		}
		if !tt.wantErr && err != nil {
			t.Fatalf("%s: expected no error, got %v", tt.name, err)
		}
	}
}
