package game

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const maxNetworkDiagnosticBytes = 1 << 20

var (
	networkDiagnosticOnce sync.Once
	networkDiagnosticMu   sync.Mutex
	networkDiagnosticPath string
)

func initNetworkDiagnostic() {
	dir, err := persistentDataDir()
	if err != nil || dir == "" {
		dir = "data"
	}
	_ = os.MkdirAll(dir, 0o755)
	networkDiagnosticPath = filepath.Join(dir, "network.log")
	if info, statErr := os.Stat(networkDiagnosticPath); statErr == nil && info.Size() > maxNetworkDiagnosticBytes {
		previous := filepath.Join(dir, "network.previous.log")
		_ = os.Remove(previous)
		_ = os.Rename(networkDiagnosticPath, previous)
	}
	appendNetworkDiagnosticRaw("=== Gun Mayhem network process pid=%d started=%s ===\n", os.Getpid(), time.Now().Format(time.RFC3339))
}

func appendNetworkDiagnosticRaw(format string, args ...any) {
	if networkDiagnosticPath == "" {
		return
	}
	f, err := os.OpenFile(networkDiagnosticPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(f, format, args...)
	_ = f.Close()
}

func networkDiagnostic(format string, args ...any) {
	networkDiagnosticOnce.Do(initNetworkDiagnostic)
	networkDiagnosticMu.Lock()
	defer networkDiagnosticMu.Unlock()
	appendNetworkDiagnosticRaw("%s "+format+"\n", append([]any{time.Now().Format("15:04:05.000")}, args...)...)
}
