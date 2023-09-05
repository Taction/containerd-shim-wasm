IMAGE_NAME ?= k3swithshims
CLUSTER_NAME ?= wasm-cluster
ARCH ?= x86_64
TARGET ?= $(ARCH)-unknown-linux-musl
TEST_IMG_NAME_SPIN ?= wasmtest_spin:latest
TEST_IMG_NAME_SLIGHT ?= wasmtest_slight:latest

compile-musl-dapr:
	make build-dapr -C ../

move-musl-to-tmp: compile-musl-dapr
	mkdir -p ./.tmp
	cp ../containerd-shim-dapr-v1 ./.tmp/

#build-multi-k3d-image: move-musl-to-tmp
#	docker buildx build -t $(IMAGE_NAME) --platform linux/amd64,linux/arm64 .

build-dev-k3d-image: move-musl-to-tmp
	docker buildx build -t $(IMAGE_NAME) --load --platform linux/amd64 .

create-k3d: build-dev-k3d-image
	k3d cluster create $(CLUSTER_NAME) --image $(IMAGE_NAME) --api-port 6550 -p "8081:8080@loadbalancer" --agents 1

up: create-k3d
	kubectl label nodes k3d-wasm-cluster-server-0 dapr-enabled=true
	kubectl apply -f ./workload

test:
	curl localhost:8081/dapr/hello

clean: 
	k3d cluster delete $(CLUSTER_NAME)

install-k3d:
	wget -q -O - https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

.PHONY: deploy clean test build-image install-k3d up compile-musl move-musl-to-tmp build-multi-k3d-image build-dev-k3d-image build-workload-images load-workload-images
