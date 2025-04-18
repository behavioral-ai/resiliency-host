package main

import (
	"context"
	"fmt"
	"github.com/behavioral-ai/collective/eventing"
	"github.com/behavioral-ai/resiliency/endpoint"
	"github.com/behavioral-ai/resiliency/operations"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"time"
)

const (
	portKey                 = "PORT"
	addr                    = "0.0.0.0:8081"
	writeTimeout            = time.Second * 300
	readTimeout             = time.Second * 15
	idleTimeout             = time.Second * 60
	healthLivelinessPattern = "/health/liveness"
	healthReadinessPattern  = "/health/readiness"

	agentsConfigPath  = "file://[cwd]/resource/agents-config.txt"
	originConfigPath  = "file://[cwd]/resource/origin-config.txt"
	loggingConfigPath = "file://[cwd]/resource/logging-operators.json"
)

func main() {
	//os.Setenv(portKey, "0.0.0.0:8082")
	port := os.Getenv(portKey)
	if port == "" {
		port = addr
	}
	start := time.Now()
	displayRuntime(port)
	handler, ok := startup(http.NewServeMux(), os.Args)
	if !ok {
		os.Exit(1)
	}
	fmt.Println(fmt.Sprintf("started : %v", time.Since(start)))
	srv := http.Server{
		Addr: port,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: writeTimeout,
		ReadTimeout:  readTimeout,
		IdleTimeout:  idleTimeout,
		Handler:      handler,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		} else {
			log.Printf("HTTP server Shutdown")
		}
		close(idleConnsClosed)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
	<-idleConnsClosed
}

func displayRuntime(port string) {
	fmt.Printf("addr    : %v\n", port)
	fmt.Printf("vers    : %v\n", runtime.Version())
	fmt.Printf("os      : %v\n", runtime.GOOS)
	fmt.Printf("arch    : %v\n", runtime.GOARCH)
	fmt.Printf("cpu     : %v\n", runtime.NumCPU())
	//fmt.Printf("env     : %v\n", core.EnvStr())
}

func startup(r *http.ServeMux, cmdLine []string) (http.Handler, bool) {
	// Initialize eventing notifier and activity
	operations.ConfigureEventing(eventing.OutputNotify, eventing.OutputActivity)
	err := operations.ConfigureLogging(loggingConfigPath, originConfigPath)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	err = operations.ConfigureAgents(agentsConfigPath, "")
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	r.Handle("/favicon.ico", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	// Initialize health handlers
	r.Handle(healthLivelinessPattern, endpoint.Health) //http.HandlerFunc(healthLivelinessHandler))
	r.Handle(healthReadinessPattern, endpoint.Health)  //http.HandlerFunc(healthReadinessHandler))

	// Operations and default http handler
	r.Handle(endpoint.OperationsPattern, endpoint.Operations) //http.HandlerFunc(endpoint.Operations.Exchange))
	r.Handle(endpoint.RootPattern, endpoint.Root)             //http.HandlerFunc(endpoint.Root.Exchange))
	return r, true
}

/*
func healthLivelinessHandler(w http.ResponseWriter, r *http.Request) {
	writeHealthResponse(w, messaging.StatusOK())
}

func healthReadinessHandler(w http.ResponseWriter, r *http.Request) {
	writeHealthResponse(w, messaging.StatusOK())
}

func writeHealthResponse(w http.ResponseWriter, status *messaging.Status) {
	if status.OK() {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("up"))

	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}


*/
