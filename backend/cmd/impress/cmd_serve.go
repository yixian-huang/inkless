package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

func serveCmd() *cobra.Command {
	var envFile string
	var port string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Impress CMS development server",
		Long:  "Start the backend server. Loads .env file if present. Override with flags or env vars.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load .env file if it exists
			if envFile != "" {
				if err := loadEnvFile(envFile); err != nil {
					return fmt.Errorf("failed to load env file %s: %w", envFile, err)
				}
			} else if _, err := os.Stat(".env"); err == nil {
				loadEnvFile(".env")
			}

			// Apply flag overrides
			if port != "" {
				os.Setenv("PORT", port)
			}

			// Set sensible defaults if not configured
			setDefault("PORT", "8088")
			setDefault("DB_DSN", "file:./data/impress.db?cache=shared&mode=rwc")
			setDefault("JWT_SECRET", "dev-jwt-secret-change-in-production")
			setDefault("JWT_REFRESH_SECRET", "dev-jwt-refresh-secret-change-in-production")
			setDefault("ENV", "development")
			setDefault("UPLOAD_DIR", "./uploads")

			// Find the server binary
			serverBin := findServerBinary()
			if serverBin == "" {
				return fmt.Errorf("server binary not found. Build it first:\n  cd backend && go build -o server ./cmd/server/")
			}

			fmt.Printf("Starting Impress CMS on port %s...\n", os.Getenv("PORT"))

			// Run the server binary with the current environment
			proc := exec.Command(serverBin)
			proc.Env = os.Environ()
			proc.Stdout = os.Stdout
			proc.Stderr = os.Stderr

			if err := proc.Start(); err != nil {
				return fmt.Errorf("failed to start server: %w", err)
			}

			// Forward signals for graceful shutdown
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				sig := <-sigCh
				proc.Process.Signal(sig)
			}()

			return proc.Wait()
		},
	}

	cmd.Flags().StringVar(&envFile, "env-file", "", "Path to .env file (default: .env in current directory)")
	cmd.Flags().StringVarP(&port, "port", "p", "", "Override server port")

	return cmd
}

func loadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Don't override existing env vars
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

func setDefault(key, value string) {
	if os.Getenv(key) == "" {
		os.Setenv(key, value)
	}
}

func findServerBinary() string {
	// Check common locations
	candidates := []string{
		"./server",
		"../server",
		filepath.Join(".", "cmd", "server", "server"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}
