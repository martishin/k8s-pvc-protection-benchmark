package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RunInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pvcbench_run_info",
		Help: "1 if a benchmark run is currently active",
	}, []string{"scenario", "pvc_size", "replicas"})

	TotalDuration = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pvcbench_total_duration_seconds",
		Help: "Total duration of the last benchmark run",
	}, []string{"scenario", "pvc_size", "replicas"})

	PVCDeleteLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pvcbench_pvc_delete_latency_seconds",
		Help:    "Latency from PVC deletion timestamp to actual deletion",
		Buckets: prometheus.DefBuckets,
	}, []string{"scenario", "pvc_size", "replicas", "ns_group"})

	ErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pvcbench_errors_total",
		Help: "Total number of errors during benchmark",
	}, []string{"type"})

	PodsRemaining = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pvcbench_progress_pods_remaining",
		Help: "Number of pods remaining to be deleted/processed",
	})

	PVCsTerminating = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pvcbench_progress_pvcs_terminating",
		Help: "Number of PVCs currently in terminating state",
	})
)

var Registry = prometheus.NewRegistry()

func init() {
	Registry.MustRegister(RunInfo)
	Registry.MustRegister(TotalDuration)
	Registry.MustRegister(PVCDeleteLatency)
	Registry.MustRegister(ErrorsTotal)
	Registry.MustRegister(PodsRemaining)
	Registry.MustRegister(PVCsTerminating)
}
