# PVC Protection Controller Load Testing Tool (`pvcbench`)

A high-performance benchmarking tool for the Kubernetes PVC protection controller (`kubernetes.io/pvc-protection`) using
realistic StatefulSet-based workloads.

## Overview

The `pvcbench` tool is designed to stress the Kubernetes PVC protection controller by creating large numbers of PVCs (
via StatefulSets) and measuring the latency between Pod deletion and actual PVC removal. This tool specifically uses
`volumeClaimTemplates` and `persistentVolumeClaimRetentionPolicy` to ensure PVCs are managed by the cluster's
controllers, rather than manual deletion.

## Prerequisites

- **Go 1.24+**
- **Minikube**
- **kubectl** configured to your minikube cluster
- **Helm** (for monitoring stack)

## 1. Minikube Setup

Start minikube with sufficient resources. A multi-node setup is recommended to better simulate a real-world cluster:

```bash
minikube start --kubernetes-version=v1.30.11 --cpus=4 --memory=8192 --nodes=3 \
  --extra-config=kubelet.max-pods=250 \
  --addons=default-storageclass \
  --addons=storage-provisioner \
  --extra-config=controller-manager.bind-address=0.0.0.0 \
  --extra-config=scheduler.bind-address=0.0.0.0 \
  --extra-config=etcd.listen-metrics-urls=http://0.0.0.0:2381
# Ensure you are in the minikube context
kubectl config use-context minikube
```

Confirm the default StorageClass exists (minikube provides one by default):

```bash
kubectl get storageclass
```

## 2. Monitoring Setup

We use `kube-prometheus-stack` to scrape metrics from the cluster control plane (kube-controller-manager,
kube-apiserver) and the `pvcbench` tool itself.

### Install Prometheus Stack

1. Add Helm repo:
   ```bash
   helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
   helm repo update
   ```

2. Install with minikube-specific values (enables control-plane scraping) into the `monitoring` namespace:
   ```bash
   kubectl create namespace monitoring
   helm install prometheus prometheus-community/kube-prometheus-stack \
     -n monitoring \
     -f deploy/monitoring/kube-prometheus-stack-values-minikube.yaml
   ```

The values file sets a 5s scrape interval for faster visibility during short benchmarks.

### Configure Scraping for `pvcbench`

The values file scrapes `host.docker.internal:8080`, which works on macOS with the Docker driver. If you are on Linux
or using another driver, update the `additionalScrapeConfigs` target to one of:

- `host.minikube.internal:8080` (recent minikube versions)
- The value of `minikube ip` with an exposed host port

### Import Dashboards

1. Import dashboards from the `dashboards/` directory:
   ```bash
   kubectl create configmap pvcbench-dashboards \
     -n monitoring \
     --from-file=dashboards/pvc-protection.json \
     --from-file=dashboards/apiserver-pressure.json \
     --from-file=dashboards/run-timeline.json
   ```
2. Label created ConfigMap:
   ```bash
   kubectl label configmap pvcbench-dashboards -n monitoring grafana_dashboard=1
   ```

### Configure Port Forwarding for Grafana

1. Port-forward Grafana:
   ```bash
   kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80
   # Default login: admin / admin
   ```

2. Open Grafana: http://localhost:3000 using `admin` / `admin` credentials

3. Dashboards to open:
    - `PVC Protection Controller Performance`: Monitor controller workqueue and PVC latency.
    - `API Server Pressure`: Monitor API server LIST QPS and latency.
    - `Run Timeline`: Correlate tool phases with cluster behavior.

4. (Optional) Port-forward Prometheus for raw PromQL testing:
   ```bash
   kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090
   ```

5. (Optional) Test that Prometheus has access to all the targets: http://localhost:9090/targets

## 3. Tool Usage

### Benchmark Commands

#### `benchmark`

Runs a single scenario in an isolated namespace (`pvcbench-<timestamp>`).

```bash
# Staggered: scale down in batches with a pause between steps
go run ./cmd/pvcbench benchmark --scenario staggered --replicas 100 --delete-batch-size 10 --delete-interval 5s

# Burst: scale from 100 to 0 immediately (worst-case controller load)
go run ./cmd/pvcbench benchmark --scenario burst --replicas 100 --pvc-size 100Mi
```

Each scenario creates a StatefulSet (pods + PVCs) first, waits for readiness, then applies the chosen scale-down pattern
and polls PVCs with GET requests (default 100ms interval) to measure delete latency.

#### `cleanup`

Deletes all benchmark namespaces created by the tool (prefixed `pvcbench-`).

```bash
go run ./cmd/pvcbench cleanup

# Force deletion by removing finalizers (use with caution):
go run ./cmd/pvcbench cleanup --force
```

### Makefile Shortcuts

Use the Makefile for quick runs:

```bash
make benchmark-burst
make benchmark-staggered
make benchmark-suite
make cleanup-benchmark-namespaces
make test
```

Override defaults with variables:

```bash
make benchmark-burst REPLICAS=50 PVC_SIZE=10Mi
make benchmark-staggered REPLICAS=80 DELETE_BATCH_SIZE=20 DELETE_INTERVAL=3s PVC_POLL_INTERVAL=250ms
```

## 4. Reviewing Results

### Console Summary

After each run, the tool prints a summary of recorded PVC deletion latencies:

```
=== Benchmark Summary ===
Total Duration: 45s
Scenario: burst
Replicas: 100
PVC Size: 100Mi
Kubernetes Version: v1.30.11
PVC Poll Interval: 500ms
PVC Delete Latency:
  Count: 100
  Avg:   2.3s
  p50:   1.8s
  p90:   4.1s
  p99:   5.8s
==========================
```

### Metrics Review (Grafana)

- **PVC Delete Latency**: Look for spikes in p99 latency during scale-down.
- **Controller Workqueue Depth**: High depth indicates the PVC protection controller is falling behind.
- **LIST Pods QPS**: The controller performs `LIST pods` frequently. Watch for spikes during scale-down.
- **API Server Latency**: High LIST latency suggests the API server is struggling under the load.

You can also validate that the metrics endpoint is live:

```bash
curl http://localhost:8080/metrics
```

## Safety Guards

- The tool will **only** run if your current kubectl context is `minikube`. This prevents accidental execution on
  production clusters.
- High QPS/Burst settings for the K8s client are configurable via flags (`--client-qps`, `--client-burst`).

## Direct kubectl Cleanup

You can also clean up manually by deleting the tool's namespaces:

```bash
kubectl delete namespace pvcbench-<timestamp>
```

## Possible Issues and Solutions

If you want to confirm the PVC protection finalizer is present during deletions:

```bash
kubectl get pvc -n pvcbench-<timestamp> -o jsonpath='{.items[0].metadata.name}{"\n"}'
kubectl get pvc <pvc-name> -n pvcbench-<timestamp> -o jsonpath='{.metadata.finalizers}{"\n"}'
```
