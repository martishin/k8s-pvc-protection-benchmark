package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartMetricsServer(port int, namespace string) {
	http.Handle("/metrics", promhttp.HandlerFor(Registry, promhttp.HandlerOpts{}))
	go func() {
		fmt.Printf("Starting metrics server on :%d\n", port)
		if namespace != "" {
			fmt.Printf("Benchmark namespace: %s\n", namespace)
		}
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			fmt.Printf("Metrics server error: %v\n", err)
		}
	}()
}
