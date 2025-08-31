package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jontolof/xcode-build-mcp/internal/mcp"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	var (
		transport   = flag.String("transport", "stdio", "Transport protocol (stdio)")
		logLevel    = flag.String("log-level", getEnvOrDefault("MCP_LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
		showVersion = flag.Bool("version", false, "Print version information")
	)
	flag.Parse()

	if *showVersion {
		log.Printf("Xcode Build MCP Server %s (built: %s)", version, buildTime)
		os.Exit(0)
	}

	logger := setupLogger(*logLevel)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	server, err := mcp.NewServer(logger)
	if err != nil {
		logger.Fatalf("Failed to create MCP server: %v", err)
	}

	go func() {
		<-sigChan
		logger.Println("Received shutdown signal")
		cancel()
	}()

	if err := server.Run(ctx, *transport); err != nil {
		logger.Fatalf("Server failed: %v", err)
	}

	// Only log shutdown in debug mode
	if *logLevel == "debug" {
		logger.Println("Server shutdown complete")
	}
}

func setupLogger(level string) *log.Logger {
	logger := log.New(os.Stderr, "[xcode-build-mcp] ", log.LstdFlags)
	
	switch level {
	case "debug":
		logger.SetFlags(log.LstdFlags | log.Lshortfile)
	case "error":
		logger.SetOutput(&errorOnlyWriter{os.Stderr})
	}
	
	return logger
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

type errorOnlyWriter struct {
	writer *os.File
}

func (w *errorOnlyWriter) Write(p []byte) (n int, err error) {
	return w.writer.Write(p)
}