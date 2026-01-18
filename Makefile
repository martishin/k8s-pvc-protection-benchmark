.PHONY: benchmark-burst benchmark-staggered benchmark-suite cleanup-benchmark-namespaces test help

PVCBENCH := go run ./cmd/pvcbench

REPLICAS ?= 100
PVC_SIZE ?= 100Mi
DELETE_BATCH_SIZE ?= 10
DELETE_INTERVAL ?= 5s
PVC_POLL_INTERVAL ?= 100ms


benchmark-burst: ## Burst: scale from N to 0 immediately (worst-case controller load).
	$(PVCBENCH) benchmark --scenario burst --replicas $(REPLICAS) --pvc-size $(PVC_SIZE) --pvc-poll-interval $(PVC_POLL_INTERVAL)

benchmark-staggered: ## Staggered: scale down in batches with an interval between steps.
	$(PVCBENCH) benchmark --scenario staggered --replicas $(REPLICAS) --pvc-size $(PVC_SIZE) \
		--delete-batch-size $(DELETE_BATCH_SIZE) --delete-interval $(DELETE_INTERVAL) --pvc-poll-interval $(PVC_POLL_INTERVAL)

benchmark-suite: ## Run burst, then staggered sequentially.
	$(MAKE) benchmark-burst
	$(MAKE) benchmark-staggered

cleanup-benchmark-namespaces: ## Delete all pvcbench-* namespaces.
	$(PVCBENCH) cleanup

test: ## Run Go tests.
	go test ./...

help: ## Show available targets and descriptions.
	@awk 'BEGIN {FS=":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "%-22s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
