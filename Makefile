APP_NAME=containerd-shim-wasm-v1

build:
	CGO_ENABLED=0 go build -ldflags "-s -w" -o $(APP_NAME) cmd/$(APP_NAME)/main.go

build-dapr:
	#disable cgo
	CGO_ENABLED=0 go build -ldflags "-s -w" -o containerd-shim-dapr-v1 cmd/$(APP_NAME)/main.go
# build go debug binary
debug:
	go build -gcflags "all=-N -l" -o $(APP_NAME) cmd/$(APP_NAME)/main.go