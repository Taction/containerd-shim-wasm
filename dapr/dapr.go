package dapr

import (
	"context"
	"fmt"
	"github.com/dmcgowan/containerd-wasm/dapr/utils"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	host_http "github.com/taction/wit-dapr/pkg/imports/host-http"
	host_state "github.com/taction/wit-dapr/pkg/imports/host-state"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"gopkg.in/yaml.v3"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
)

type Config struct {
	Port          int    `yaml:"port" json:"port"`
	WASMPath      string `yaml:"wasmPath" json:"wasmPath"`
	ComponentPath string `yaml:"componentPath" json:"componentPath"`
	Trigger       string `yaml:"trigger" json:"trigger"`
}

type Runtime struct {
	finish          chan struct{}
	close           chan struct{}
	rootfs          string
	config          *Config
	instanceCounter atomic.Uint64
	runtimeConfig   wazero.ModuleConfig

	runtime    wazero.Runtime
	module     wazero.CompiledModule
	moduleName string
}

func NewRuntime(rootfs string, c *Config, runtimeConfig wazero.ModuleConfig) *Runtime {
	return &Runtime{rootfs: rootfs, config: c, runtimeConfig: runtimeConfig, finish: make(chan struct{}), close: make(chan struct{})}
}

func (rt *Runtime) Run() error {
	if err := rt.initFromConfig(); err != nil {
		return err
	}
	err := rt.run()
	if err != nil {
		return err
	}

	return nil
}

func (rt *Runtime) Wait() {
	<-rt.finish
}

func (rt *Runtime) initFromConfig() error {
	wasmFile := rt.config.WASMPath
	wasmName := filepath.Base(wasmFile)
	wasmCode, err := os.ReadFile(filepath.Join(rt.rootfs, wasmFile))
	if err != nil {
		pwd, _ := os.Getwd()
		// read all files and dirs from pwd
		var files string
		filepath.Walk(pwd, func(path string, info os.FileInfo, err error) error {
			files += fmt.Sprintf("%s, ", info.Name())
			return nil
		})
		return fmt.Errorf("could not read WASM file '%s' with rootfs '%s' in '%s' (existing file/dir: %s) err: %w", wasmFile, rt.rootfs, pwd, files, err)
	}
	ctx := context.Background()
	runtime := wazero.NewRuntime(ctx)

	// todo detect wasm module valid
	wasmModule, err := runtime.CompileModule(ctx, wasmCode)
	if err != nil {
		return err
	}

	// todo detect wasi p1
	wasi_snapshot_preview1.Instantiate(ctx, runtime)
	if rt.config.Trigger == "state" {
		host_state.Instantiate(ctx, runtime)
	}
	rt.runtime = runtime
	rt.module = wasmModule
	rt.moduleName = wasmName
	//rt.runtimeConfig = rt.runtimeConfig.WithName(wasmName).WithSysNanotime().WithSysWalltime().WithSysNanosleep()
	return nil
}

func (rt *Runtime) Close() error {
	err := rt.runtime.Close(context.TODO())
	close(rt.close)
	return err
}

func LoadConfigFromFile(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		logrus.Warnf("load config error when reading file %s : %s", path, err)
		return nil, err
	}
	c := &Config{}
	err = yaml.Unmarshal(b, c)
	return c, err
}

func (rt *Runtime) run() error {
	if rt.config.Trigger == "http" {
		return rt.runServer()
	} else {
		return rt.runExecWasm()
	}
}

func (rt *Runtime) runServer() error {
	errch := make(chan error, 1)
	log.Printf("Listening on http://localhost:%d", rt.config.Port)
	go utils.StartServer(rt.config.Port, rt.appRouter(), true, false, rt.close, errch)
	go func() {
		err, ok := <-errch
		if ok {
			fmt.Println(err)
		}
		close(rt.finish)
	}()
	return nil
}

func (rt *Runtime) runExecWasm() error {
	if len(rt.config.ComponentPath) > 0 {
		comps, err := LoadComponents(filepath.Join(rt.rootfs, rt.config.ComponentPath))
		if err != nil {
			return err
		}
		host_state.AddComp(comps...)
	}

	ins, err := rt.runtime.InstantiateModule(context.TODO(), rt.module, rt.runtimeConfig)
	if err != nil {
		return err
	}
	ins.Close(context.Background())
	close(rt.finish)
	return nil
}

func (rt *Runtime) IndexHandler(w http.ResponseWriter, r *http.Request) {
	id := rt.instanceCounter.Add(1)
	c := rt.runtimeConfig.WithName(fmt.Sprintf("%s-%d", rt.moduleName, id)).WithSysNanotime().WithSysWalltime().WithSysNanosleep()
	ins, err := rt.runtime.InstantiateModule(context.TODO(), rt.module, c)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("instantiation wasm error: " + err.Error()))
		ins.Close(context.TODO())
		return
	}
	ws := host_http.WasmServer{Module: ins}
	ws.ServeHTTP(w, r)
}

// appRouter initializes restful api router
func (rt *Runtime) appRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	router.PathPrefix("/").Handler(http.HandlerFunc(rt.IndexHandler))
	//router.HandleFunc("/healthz", rt.HealthHandler).Methods("GET", "POST")
	router.Use(mux.CORSMethodMiddleware(router))

	return router
}

// In the future
const (
	modeDefault importMode = iota
	modeWasiP1
	modeState
)

type importMode uint

func detectImports(imports []api.FunctionDefinition) map[importMode]bool {
	result := make(map[importMode]bool)
	for _, f := range imports {
		moduleName, _, _ := f.Import()
		switch moduleName {
		case wasi_snapshot_preview1.ModuleName:
			result[modeWasiP1] = true
		case host_state.ModuleName:
			result[modeState] = true
		}
	}
	return result
}
