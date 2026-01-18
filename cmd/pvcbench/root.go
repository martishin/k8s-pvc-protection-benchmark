package main

import (
	"fmt"
	"os"

	"pvc-protection-bench/pkg/logging"

	"github.com/spf13/cobra"
)

var (
	clientQPS   float32
	clientBurst int
	metricsPort int
)

var rootCmd = &cobra.Command{
	Use:   "pvcbench",
	Short: "PVC Protection Controller Load Testing Tool",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logging.InitLogger()
	},
}

func init() {
	rootCmd.PersistentFlags().Float32Var(&clientQPS, "client-qps", 200, "Kubernetes client QPS")
	rootCmd.PersistentFlags().IntVar(&clientBurst, "client-burst", 400, "Kubernetes client Burst")
	rootCmd.PersistentFlags().IntVar(&metricsPort, "metrics-port", 8080, "Port for Prometheus metrics")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
