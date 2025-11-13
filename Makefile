# -------- Config --------
APP            ?= zero-downtime
NAMESPACE      ?= default

# Always use this repo name
IMAGE_REPO     := varsilias/zero-downtime

# Build info (ldflags)
VERSION        ?= $(shell git rev-parse --short HEAD)-$(shell date -u +%Y%m%d%H%M%S)
COMMIT         ?= $(shell git rev-parse --short HEAD)
BUILT_AT       ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

IMAGE          := $(IMAGE_REPO):$(VERSION)

# K8s
GCP_CONTEXT 	= zero-downtime-gcp-cluster
K_DEPLOY        = k8s/deployment.yaml
K_SERVICE       = k8s/service.yaml
K_PVC			= k8s/pvc.yaml
K_INGRESS		= k8s/ingress.yaml

# -------- Targets --------
.PHONY: release docker-build docker-push set-gcp-context load-minikube apply set-image rollout url logs status history undo restart

release-gcp: docker-build docker-push set-gcp-context apply set-image rollout url ## Build, push, set-context, deploy, wait, print URL

release: docker-build load-minikube apply set-image rollout url ## Build, load, deploy, wait, print URL

docker-build: ## Build container image with linker flags (Dockerfile handles Node/Tailwind)
	@echo ">>> Building image: $(IMAGE)"
	docker build \
		--build-arg VERSION="$(VERSION)" \
		--build-arg COMMIT="$(COMMIT)" \
		--build-arg BUILT_AT="$(BUILT_AT)" \
		-t "$(IMAGE)" .

load-minikube: ## Load image into Minikube's container runtime
	@echo ">>> Loading image into Minikube: $(IMAGE)"
	minikube image load "$(IMAGE)"

docker-push: ## Load image into Minikube's container runtime
	@echo ">>> Pushing image to DockerHub: $(IMAGE)"
	docker push "$(IMAGE)"

set-gcp-context:
	@echo ">>> Setting kubectl context to GKE Cluster : $(GCP_CONTEXT)"
	kubectl config use-context "$(GCP_CONTEXT)"

ingress: ## Show a ingress URL to reach the service
	@echo ">>> Get URL(s):"
	@kubectl -n "$(NAMESPACE)" get ing $(APP) -o jsonpath='{.spec.rules[0].host}'

apply: ## Apply service and deployment manifests
	@echo ">>> Applying manifests to namespace $(NAMESPACE)"
	kubectl -n "$(NAMESPACE)" apply -f "$(K_PVC)"
	kubectl -n "$(NAMESPACE)" apply -f "$(K_SERVICE)"
	kubectl -n "$(NAMESPACE)" apply -f "$(K_DEPLOY)"
	kubectl -n "$(NAMESPACE)" apply -f "$(K_INGRESS)"

set-image: ## Point deployment to the freshly built image
	@echo ">>> Setting image on deployment/$(APP) -> $(IMAGE)"
	kubectl -n "$(NAMESPACE)" set image deploy/$(APP) "$(APP)"="$(IMAGE)"

rollout: ## Wait for rolling update to finish
	@echo ">>> Waiting for rollout to complete"
	kubectl -n "$(NAMESPACE)" rollout status deploy/$(APP)""

url: ## Show a local URL to reach the service
	@echo ">>> Service URL(s):"
	@minikube service $(APP) --url -n "$(NAMESPACE)"

# -------- Handy ops --------
logs: ## Tail logs
	kubectl -n "$(NAMESPACE)" logs -l app=$(APP) -f --all-containers --max-log-requests=6

status: ## Pods status
	kubectl -n "$(NAMESPACE)" get pods -l app=$(APP) -o wide

history: ## Rollout history
	kubectl -n "$(NAMESPACE)" rollout history deploy/$(APP)

undo: ## Roll back
	kubectl -n "$(NAMESPACE)" rollout undo deploy/$(APP)

restart: ## Force rolling restart
	kubectl -n "$(NAMESPACE)" rollout restart deploy/$(APP)
