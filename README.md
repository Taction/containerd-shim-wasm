## containerd wasm shim

**PROOF OF CONCEPT WARNING**

A wasm implementation of a containerd runtime using the containerd shim interface

Uses wasmer to execute wasm/wasi binaries

## debug

    ```sh
    make debug
    cp containerd-shim-wasm-v1 /usr/local/bin/
    ```
build image
```shell
docker build -t docker4zc/wasmstring:latest .
docker push docker4zc/wasmstring:latest
ctr images pull docker.io/docker4zc/wasmstring:latest
```

debug
```shell
export CONTAINERD_SHIM_RUNHCS_V1_WAIT_DEBUGGER="true"
# you can check by running echo $CONTAINERD_SHIM_RUNHCS_V1_WAIT_DEBUGGER
ctr run --rm --runtime=io.containerd.wasm.v1 docker.io/docker4zc/wasmstring:latest testwasm
```




## references
https://www.jamessturtevant.com/posts/attaching-a-debugger-to-windows-containerd-shim/