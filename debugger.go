package wasm

import (
	"os"
	"os/signal"
	"syscall"
)

// setupDebuggerEvent listens for an user1 signal to allow a debugger such as delve
// to attach for advanced debugging. It's called when handling a ContainerCreate
func setupDebuggerEvent() {
	//logrus.Infof("enter setupDebuggerEvent with CONTAINERD_SHIM_WASM_V1_WAIT_DEBUGGER=%s", os.Getenv("CONTAINERD_SHIM_WASM_V1_WAIT_DEBUGGER"))
	//logrus.Infof("enter setupDebuggerEvent print enviroment %v", os.Environ())

	if os.Getenv("CONTAINERD_SHIM_WASM_V1_WAIT_DEBUGGER") == "" {
		return
	}

	// wait user1 signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1)
	<-c
	//logrus.Infof("received SIGUSR1, continue...")
}
