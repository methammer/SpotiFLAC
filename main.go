package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/afkarxyz/SpotiFLAC/backend"
)

const port = "6890"

func main() {
	// ── Config dir ────────────────────────────────────────────────────────
	configDir, err := getConfigDir()
	if err != nil {
		fmt.Printf("FATAL: cannot determine config dir: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("FATAL: cannot create config dir: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[Main] Config dir: %s\n", configDir)


	// ── Job manager (workers + BoltDB) ────────────────────────────────────
	if err := InitJobManager(configDir); err != nil {
		fmt.Printf("FATAL: cannot init job manager: %v\n", err)
		os.Exit(1)
	}
        // ── History DB (shared avec jobs.db) ─────────────────────────────────
        if err := backend.InitHistoryDBShared(GetJobManager().db); err != nil {
                fmt.Printf("[Main] Warning: failed to init history buckets: %v\n", err)
        }
	defer CloseJobManager()

	// ── Auth (Jellyfin + JWT) ───────────────────────────────────────────────
	if err := InitAuth(GetJobManager().db); err != nil {
		fmt.Printf("FATAL: cannot init auth: %v\n", err)
		os.Exit(1)
	}
	// ── Watcher (playlist sync) ───────────────────────────────────────────
	InitWatcher(GetJobManager())
	defer CloseWatcher()

	// ── App + HTTP server ─────────────────────────────────────────────────
	app := NewApp()
	app.startup(context.Background())

	server := NewServer(app)

	httpServer := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      server,
		ReadTimeout:  0,            // pas de timeout — les downloads peuvent être longs
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	// ── Graceful shutdown ─────────────────────────────────────────────────
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Printf("[Main] SpotiFLAC listening on http://0.0.0.0:%s\n", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("FATAL: server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-stop
	fmt.Println("[Main] Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)

	app.shutdown(context.Background())
	fmt.Println("[Main] Bye.")
}

// getConfigDir retourne le dossier de config SpotiFLAC.
// Sous Docker : /home/nonroot/.SpotiFLAC
// En local    : ~/.SpotiFLAC
func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".SpotiFLAC"), nil
}
