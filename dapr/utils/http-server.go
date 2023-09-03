/*
Copyright 2022 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	tlsCertEnvKey = "TLS_CERT"
	tlsKeyEnvKey  = "TLS_KEY"
)

// StartServer starts a HTTP or HTTP2 server
func StartServer(port int, appRouter http.Handler, allowHTTP2 bool, enableTLS bool, closeCh <-chan struct{}, errCh chan<- error) {

	// Create a listener
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}

	var server *http.Server
	if allowHTTP2 {
		// Create a server capable of supporting HTTP2 Cleartext connections
		// Also supports HTTP1.1 and upgrades from HTTP1.1 to HTTP2
		h2s := &http2.Server{}
		//nolint:gosec
		server = &http.Server{
			Addr:    addr,
			Handler: h2c.NewHandler(appRouter, h2s),
		}
	} else {
		//nolint:gosec
		server = &http.Server{
			Addr:    addr,
			Handler: appRouter,
		}
	}

	var certFile, keyFile string
	if enableTLS {
		certFile, keyFile, err = getTLSCertAndKey()
		if err != nil {
			log.Fatalf("Failed to get TLS cert and key: %v", err)
		}
	}

	// Stop the server when we get a termination signal
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT) //nolint:staticcheck
	go func() {
		// Wait for cancelation signal
		select {
		case <-closeCh:
		case <-stopCh:
		}
		//<-stopCh
		log.Println("Shutdown signal received")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// Blocking call
	if enableTLS {
		err = server.ServeTLS(ln, certFile, keyFile)
	} else {
		err = server.Serve(ln)
	}

	if err != http.ErrServerClosed {
		log.Printf("Failed to run server: %v\n", err)
		errCh <- err
	}
	close(errCh)
	log.Println("Server shut down")
}

func getTLSCertAndKey() (string, string, error) {
	cert, ok := os.LookupEnv(tlsCertEnvKey)
	if !ok {
		return "", "", fmt.Errorf("%s is not set", tlsCertEnvKey)
	}
	key, ok := os.LookupEnv(tlsKeyEnvKey)
	if !ok {
		return "", "", fmt.Errorf("%s is not set", tlsKeyEnvKey)
	}
	return cert, key, nil
}
