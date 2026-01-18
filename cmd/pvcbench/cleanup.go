package main

import (
	"context"
	"fmt"
	"strings"

	"pvc-protection-bench/pkg/k8s"
	"pvc-protection-bench/pkg/logging"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	forceDelete bool
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up namespaces created by the tool",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateCleanupArgs(args); err != nil {
			return err
		}

		client, err := k8s.NewClient(clientQPS, clientBurst)
		if err != nil {
			return err
		}

		ctx := context.Background()
		logger := logging.GetLogger()

		namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, ns := range namespaces.Items {
			shouldDelete := strings.HasPrefix(ns.Name, "pvcbench-")

			if shouldDelete {
				logger.Info("deleting namespace", logging.StringField("name", ns.Name))
				if err := k8s.DeleteNamespace(ctx, client, ns.Name); err != nil {
					logger.Error("failed to delete namespace", logging.StringField("name", ns.Name), logging.ErrorField(err))
					return err
				}
				if forceDelete {
					if err := k8s.ForceDeleteNamespace(ctx, client, ns.Name); err != nil {
						logger.Error("force delete namespace failed", logging.StringField("name", ns.Name), logging.ErrorField(err))
						return err
					}
				} else {
					if err := k8s.WaitForNamespaceDeleted(ctx, client, ns.Name); err != nil {
						logger.Error("waiting for namespace deletion failed", logging.StringField("name", ns.Name), logging.ErrorField(err))
						return err
					}
				}
			}
		}

		return nil
	},
}

func init() {
	cleanupCmd.Flags().BoolVar(&forceDelete, "force", false, "Force namespace deletion by removing finalizers")
	rootCmd.AddCommand(cleanupCmd)
}

func validateCleanupArgs(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("cleanup does not accept positional arguments")
	}
	return nil
}
