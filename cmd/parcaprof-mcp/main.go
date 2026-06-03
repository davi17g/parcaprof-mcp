package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	parcaclient "github.com/davi17g/parcaprof-mcp/internal/parca"
	"github.com/davi17g/parcaprof-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Set via -ldflags at build time (see Makefile).
var (
	appVersion = "dev"
	commitHash = "none"
	buildTime  = "unknown"
)

func main() {
	var (
		parcaAddr   = flag.String("parca-address", "localhost:7070", "Parca gRPC address (host:port)")
		parcaInsec  = flag.Bool("parca-insecure", false, "use plaintext (no TLS) when connecting to Parca")
		transport   = flag.String("transport", "http", "MCP transport: 'http' (Streamable HTTP + SSE) or 'stdio'")
		httpAddr    = flag.String("http-addr", ":8080", "HTTP listen address (transport=http)")
		showVersion = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("parcaprof-mcp %s\ncommit:  %s\nbuilt:   %s\n", appVersion, commitHash, buildTime)
		return
	}

	cfg := parcaclient.Config{
		Address:     *parcaAddr,
		Insecure:    *parcaInsec,
		BearerToken: os.Getenv("PARCA_BEARER_TOKEN"),
	}
	pc, err := parcaclient.Dial(cfg)
	if err != nil {
		log.Fatalf("parca: %v", err)
	}
	srv := mcp.NewServer(&mcp.Implementation{Name: "parcaprof-mcp", Version: appVersion}, nil)
	tools.Register(srv, pc)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	exitCode := 0
	switch *transport {
	case "stdio":
		log.Printf("parcaprof-mcp %s: stdio transport, parca=%s", appVersion, *parcaAddr)
		if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil {
			log.Printf("mcp stdio: %v", err)
			exitCode = 1
		}
	case "http":
		getSrv := func(*http.Request) *mcp.Server { return srv }
		mux := http.NewServeMux()
		mux.Handle("/mcp", mcp.NewStreamableHTTPHandler(getSrv, nil))
		mux.Handle("/mcp/", mcp.NewStreamableHTTPHandler(getSrv, nil))
		mux.Handle("/sse", mcp.NewSSEHandler(getSrv, nil))
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

		hsrv := &http.Server{
			Addr:              *httpAddr,
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		}
		go func() {
			<-ctx.Done()
			_ = hsrv.Shutdown(context.Background())
		}()
		log.Printf("parcaprof-mcp %s: http on %s (/mcp streamable, /sse legacy), parca=%s", appVersion, *httpAddr, *parcaAddr)
		if err := hsrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http: %v", err)
			exitCode = 1
		}
	default:
		log.Printf("unknown transport %q (want 'http' or 'stdio')", *transport)
		exitCode = 2
	}
	stop()
	pc.Close()
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
