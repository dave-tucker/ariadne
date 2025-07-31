all: build

bin:
	@mkdir -p $@

.PHONY: build
build: | bin
	@go build -o ./bin/ovs-vswitch-mcp ./cmd/ovs-vswitch-mcp 
	@go build -o ./bin/ovn-nbdb-mcp ./cmd/ovn-nbdb-mcp
	@go build -o ./bin/ovn-sbdb-mcp ./cmd/ovn-sbdb-mcp
	@go build -o ./bin/ovn-ic-nbdb-mcp ./cmd/ovn-ic-nbdb-mcp
	@go build -o ./bin/ovn-ic-sbdb-mcp ./cmd/ovn-ic-sbdb-mcp
	
.PHONY: docker-images
docker-images: build
	@docker build --target ovs-vswitch-mcp -t network-researcher/ovsdb-mcp:latest .
	@docker build --target ovn-nbdb-mcp -t network-researcher/ovn-nbdb-mcp:latest .
	@docker build --target ovn-sbdb-mcp -t network-researcher/ovn-sbdb-mcp:latest .
	@docker build --target ovn-ic-nbdb-mcp -t network-researcher/ovn-ic-nbdb-mcp:latest .
	@docker build --target ovn-ic-sbdb-mcp -t network-researcher/ovn-ic-sbdb-mcp:latest .
	@docker build --target network-researcher -t network-researcher/network-researcher:latest .

.PHONY: load-images
load-images:
	kind load docker-image --name ovn network-researcher/ovsdb-mcp:latest
	kind load docker-image --name ovn network-researcher/ovn-nbdb-mcp:latest
	kind load docker-image --name ovn network-researcher/ovn-sbdb-mcp:latest
	kind load docker-image --name ovn network-researcher/ovn-ic-nbdb-mcp:latest
	kind load docker-image --name ovn network-researcher/ovn-ic-sbdb-mcp:latest
	kind load docker-image --name ovn network-researcher/network-researcher:latest

.PHONY: test
test:
	go test ./...

.PHONY: test-integration
test-integration:
	go test ./test/...

.PHONY: clean
clean:
	rm -rf bin _cache

ovn-kubernetes:
	git clone https://github.com/ovn-org/ovn-kubernetes.git --depth 1
	cd ovn-kubernetes

# TODO: Create a kind cluster with ovn-kubernetes installed using kind and helm
#.PHONY: kind-cluster
#kind-cluster:
#	kind create cluster --config k8s/kind.yaml
#	for n in $(kind get nodes); do kubectl label node "${n}" k8s.ovn.org/zone-name=${n} --overwrite ; done
#	helm repo add ovnk https://flavio-fernandes.github.io/ovn-kubernetes
#	helm install ovnk/ovn-kubernetes -f ./k8s/values.yaml --generate-name \
		--set k8sAPIServer="https://$(kubectl get pods -n kube-system -l component=kube-apiserver -o jsonpath='{.items[0].status.hostIP}'):6443" \
		--set global.image.repository=ghcr.io/ovn-kubernetes/ovn-kubernetes/ovn-kube-fedora \
   		--set global.image.tag=release-1.0
#
#.PHONY: run-on-kind
#run-on-kind: kind-cluster docker-images load-images

.PHONY: deploy
deploy: docker-images load-images
	kubectl apply -k ./k8s/workloads/base
	kubectl apply -f ./k8s/ariadne/namespace.yaml
	kubectl apply -k ./k8s/ariadne/overlays/dev

.PHONY: redeploy
redeploy: docker-images load-images
	@echo "Checking for existing pods in network-researcher namespace..."
	@if kubectl get pods -n network-researcher --no-headers 2>/dev/null | grep -q .; then \
		echo "Found existing pods, deleting deployment..."; \
		kubectl delete -k ./k8s/ariadne/overlays/dev; \
	else \
		echo "No existing pods found, skipping delete..."; \
	fi
	@echo "Applying new deployment..."
	@kubectl apply -k ./k8s/ariadne/overlays/dev
