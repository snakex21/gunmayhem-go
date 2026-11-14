//go:build windows

package game

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCloudflareStartupRetriesThenReportsFinalError uses where.exe as a fake
// cloudflared that exits immediately, so every attempt fails. The monitor must
// restart it up to the attempt limit and then expose one final error.
func TestCloudflareStartupRetriesThenReportsFinalError(t *testing.T) {
	fake := filepath.Join(os.Getenv("SystemRoot"), "System32", "where.exe")
	if _, err := os.Stat(fake); err != nil {
		if p, lookErr := exec.LookPath("where.exe"); lookErr == nil {
			fake = p
		} else {
			t.Skip("where.exe not available")
		}
	}
	tunnel := &cloudflareQuickTunnel{
		executable:  fake,
		origin:      "http://127.0.0.1:1",
		maxAttempts: cloudflareTunnelMaxAttempts,
	}
	defer tunnel.Close()
	if err := tunnel.startProcess(1); err != nil {
		t.Fatal(err)
	}
	tunnel.monitorStartup()

	sawRetryLabel := false
	deadline := time.Now().Add(20 * time.Second)
	for {
		_, _, _, err := tunnel.Status()
		if err != nil {
			if !strings.Contains(err.Error(), "after 3 attempts") || !strings.Contains(err.Error(), "cloudflared stopped") {
				t.Fatalf("final error lacks attempts/reason: %v", err)
			}
			break
		}
		if strings.Contains(tunnel.StartupStatusText(), "RETRY") {
			sawRetryLabel = true
		}
		if time.Now().After(deadline) {
			t.Fatal("tunnel startup never reported a final error")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if attempt, maxAttempts := tunnel.StartupAttempt(); attempt != maxAttempts {
		t.Fatalf("attempt=%d want %d", attempt, maxAttempts)
	}
	if !sawRetryLabel {
		t.Fatal("status text never showed a retry")
	}
	if got := tunnel.StartupStatusText(); got != "TUNNEL ERROR" {
		t.Fatalf("status text=%q", got)
	}
}
