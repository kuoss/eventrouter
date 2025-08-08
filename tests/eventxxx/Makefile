CLUSTER_NAME := kind1
VERSION := latest
IMG := ghcr.io/kuoss/eventrouter:$(VERSION)

.PHONY: kind-create
kind-create:
	kind create cluster --name $(CLUSTER_NAME)

.PHONY: kind-delete
kind-delete:
	kind delete cluster --name $(CLUSTER_NAME)

.PHONY: eventbit
eventbit:
	docker pull fluent/fluent-bit:latest
	kind load docker-image fluent/fluent-bit:latest --name $(CLUSTER_NAME)
	kubectl apply -f eventbit.yaml
	kubectl -n kube-system get pod -l app=eventbit
	kubectl -n kube-system rollout restart deploy -l app=eventbit
	kubectl -n kube-system logs -l app=eventbit -f

.PHONY: eventexp
eventexp:
	docker pull ghcr.io/resmoio/kubernetes-event-exporter:latest
	kind load docker-image ghcr.io/resmoio/kubernetes-event-exporter:latest --name $(CLUSTER_NAME)
	kubectl apply -f eventexp.yaml
	kubectl -n kube-system get pod -l app=eventexp
	kubectl -n kube-system rollout restart deploy -l app=eventexp
	kubectl -n kube-system logs -l app=eventexp -f
