package main

import "testing"

func TestValidateCleanupArgs(t *testing.T) {
	if err := validateCleanupArgs(nil); err != nil {
		t.Fatalf("expected no error for empty args: %v", err)
	}
	if err := validateCleanupArgs([]string{"extra"}); err == nil {
		t.Fatalf("expected error for extra args")
	}
}
