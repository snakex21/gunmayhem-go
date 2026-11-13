package networksetup

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestFirewallRulesUseCurrentPathsAndNarrowPorts(t *testing.T) {
	executablePath := filepath.Join("package", "GunMayhem-Windows-x64", "GunMayhem.exe")
	cloudflaredPath := filepath.Join(filepath.Dir(executablePath), "cloudflared.exe")
	rules := firewallRules(executablePath)
	if got, want := len(rules), 6; got != want {
		t.Fatalf("firewallRules returned %d rules, want %d", got, want)
	}

	cases := []struct {
		name       string
		direction  string
		protocol   string
		localPort  int
		remotePort int
		program    string
		remoteIP   string
	}{
		{name: ruleDiscovery, direction: "in", protocol: "UDP", localPort: lanDiscoveryPort, program: executablePath, remoteIP: "LocalSubnet"},
		{name: ruleGameplay, direction: "in", protocol: "TCP", localPort: lanGameplayPort, program: executablePath, remoteIP: "LocalSubnet"},
		{name: ruleDHT, direction: "in", protocol: "UDP", localPort: lanGameplayPort, program: executablePath, remoteIP: "any"},
		{name: ruleCloudflareTCP, direction: "out", protocol: "TCP", remotePort: cloudflareTunnelPort, program: cloudflaredPath, remoteIP: "any"},
		{name: ruleCloudflareUDP, direction: "out", protocol: "UDP", remotePort: cloudflareTunnelPort, program: cloudflaredPath, remoteIP: "any"},
		{name: ruleCloudflareHTTP, direction: "out", protocol: "TCP", remotePort: cloudflareHealthPort, program: executablePath, remoteIP: "any"},
	}

	for _, want := range cases {
		var got firewallRule
		found := false
		for _, rule := range rules {
			if rule.name == want.name {
				got = rule
				found = true
				break
			}
		}
		if !found {
			t.Errorf("rule %q was not returned", want.name)
			continue
		}
		if got.direction != want.direction || got.protocol != want.protocol ||
			got.localPort != want.localPort || got.remotePort != want.remotePort ||
			got.program != want.program || got.remoteIP != want.remoteIP {
			t.Errorf("rule %q = %+v, want direction=%q protocol=%q localPort=%d remotePort=%d program=%q remoteIP=%q",
				want.name, got, want.direction, want.protocol, want.localPort, want.remotePort, want.program, want.remoteIP)
		}
		if (got.localPort > 0) == (got.remotePort > 0) {
			t.Errorf("rule %q must specify exactly one of localport or remoteport", want.name)
		}
		if got.program == "" {
			t.Errorf("rule %q has no program scope", want.name)
		}
		args := firewallAddArgs(got)
		joinedArgs := strings.Join(args, " ")
		hasArg := func(want string) bool {
			for _, arg := range args {
				if arg == want {
					return true
				}
			}
			return false
		}
		if !hasArg("program=" + got.program) {
			t.Errorf("rule %q is missing program scope: %s", want.name, joinedArgs)
		}
		if got.direction == "in" && !hasArg("localport="+strconv.Itoa(got.localPort)) {
			t.Errorf("rule %q is missing its inbound local port: %s", want.name, joinedArgs)
		}
		if got.direction == "out" && !hasArg("remoteport="+strconv.Itoa(got.remotePort)) {
			t.Errorf("rule %q is missing its outbound remote port: %s", want.name, joinedArgs)
		}
		if got.remoteIP != "" && !hasArg("remoteip="+got.remoteIP) {
			t.Errorf("rule %q is missing remoteip=%s: %s", want.name, got.remoteIP, joinedArgs)
		}
		if strings.Contains(strings.ToLower(joinedArgs), "localport=any") ||
			strings.Contains(strings.ToLower(joinedArgs), "remoteport=any") {
			t.Errorf("rule %q unexpectedly allows all ports: %s", want.name, joinedArgs)
		}
	}
}

func TestFirewallOutputHasProgramRejectsStalePath(t *testing.T) {
	currentPath := filepath.Join("package", "current", "GunMayhem.exe")
	oldPath := filepath.Join("package", "old", "GunMayhem.exe")
	oldOutput := []byte("Rule Name: Gun Mayhem Gameplay TCP 7777\r\nProgram: " + oldPath + "\r\n")
	if firewallOutputHasProgram(oldOutput, currentPath) {
		t.Fatal("stale executable path was accepted")
	}

	currentOutput := []byte("Rule Name: Gun Mayhem Gameplay TCP 7777\r\nProgram: " + currentPath + "\r\n")
	if !firewallOutputHasProgram(currentOutput, currentPath) {
		t.Fatal("current executable path was not detected")
	}

	prefixOutput := []byte("Program: " + currentPath + ".old\r\n")
	if firewallOutputHasProgram(prefixOutput, currentPath) {
		t.Fatal("path prefix was incorrectly accepted")
	}
}
