package game

import "testing"

func TestDeveloperToolsAreOptIn(t *testing.T) {
	if g := New(); g.developerToolsEnabled {
		t.Fatal("New() enabled developer tools; normal/original builds must keep them disabled")
	}
	if g := NewWithDeveloperTools(true); !g.developerToolsEnabled {
		t.Fatal("NewWithDeveloperTools(true) did not enable developer tools")
	}
}
