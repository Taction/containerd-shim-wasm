APP_NAME=containerd-shim-wasm-v1

build:
	go build -o $(APP_NAME) cmd/$(APP_NAME)/main.go

# build go debug binary
debug:
	go build -gcflags "all=-N -l" -o $(APP_NAME) cmd/$(APP_NAME)/main.go