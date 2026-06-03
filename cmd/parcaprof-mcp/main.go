package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	parcaclient "github.com/davi17g/parcaprof-mcp/internal/parca"
	"github.com/davi17g/parcaprof-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const version = "0.1.0"

func main() {
	var (
		parcaAddr   = flag.String("parca-address", "localhost:7070", "Parca gRPC address (host:port)")
		parcaInsec  = flag.Bool("parca-insecure", false, "use plaintext (no TLS) when connecting to Parca")
		transport   = flag.String("transport", "http", "MCP transport: 'http' (Streamable HTTP + SSE) or 'stdio'")
		httpAddr    = flag.String("http-addr", ":8080", "HTTP listen address (transport=http)")
	)
	flag.Parse()

	cfg := parcaclient.Config{
		Address:     *parcaAddr,
		Insecure:    *parcaInsec,
		BearerToken: os.Getenv("PARCA_BEARER_TOKEN"),
	}
	pc, err := parcaclient.Dial(cfg)
	if err != nil {
		log.Fatalf("parca: %v", err)
	}
	defer pc.Close()

	srv := mcp.NewServer(&mcp.Implementation{Name: "parcaprof-mcp", Version: version}, nil)
	tools.Register(srv, pc)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch *transport {
	case "stdio":
		log.Printf("parcaprof-mcp %s: stdio transport, parca=%s", version, *parcaAddr)
		if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil {
			log.Fatalf("mcp stdio: %v", err)
		}
	case "http":
		getSrv := func(*http.Request) *mcp.Server { return srv }
		mux := http.NewServeMux()
		mux.Handle("/mcp", mcp.NewStreamableHTTPHandler(getSrv, nil))
		mux.Handle("/mcp/", mcp.NewStreamableHTTPHandler(getSrv, nil))
		mux.Handle("/sse", mcp.NewSSEHandler(getSrv, nil))
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

		hsrv := &http.Server{Addr: *httpAddr, Handler: mux}
		go func() {
			<-ctx.Done()
			_ = hsrv.Shutdown(context.Background())
		}()
		log.Printf("parcaprof-mcp %s: http on %s (/mcp streamable, /sse legacy), parca=%s", version, *httpAddr, *parcaAddr)
		if err := hsrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	default:
		log.Fatalf("unknown transport %q (want 'http' or 'stdio')", *transport)
	}
}
