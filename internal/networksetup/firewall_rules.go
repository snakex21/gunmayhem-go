package networksetup

import (
	"path/filepath"
	"strconv"
	"strings"
)

const (
	lanGameplayPort      = 7777
	lanDiscoveryPort     = 7778
	cloudflareTunnelPort = 7844
	cloudflareHealthPort = 443
)

const (
	ruleDiscovery      = "Gun Mayhem LAN Discovery"
	ruleGameplay       = "Gun Mayhem LAN Gameplay"
	ruleDHT            = "Gun Mayhem P2P DHT UDP 7777"
	ruleCloudflareTCP  = "Gun Mayhem Cloudflare TCP 7844"
	ruleCloudflareUDP  = "Gun Mayhem Cloudflare UDP 7844"
	ruleCloudflareHTTP = "Gun Mayhem Cloudflare Health HTTPS 443"
)

type firewallRule struct {
	name       string
	direction  string
	protocol   string
	localPort  int
	remotePort int
	program    string
	remoteIP   string
}

// firewallRules returns the complete, intentionally narrow set of rules used
// by the Windows launcher. The executable and cloudflared paths are resolved
// from the running package rather than from a build-time path.
func firewallRules(executablePath string) []firewallRule {
	executablePath = filepath.Clean(executablePath)
	cloudflaredPath := filepath.Join(filepath.Dir(executablePath), "cloudflared.exe")

	return []firewallRule{
		{
			name:      ruleDiscovery,
			direction: "in",
			protocol:  "UDP",
			localPort: lanDiscoveryPort,
			program:   executablePath,
			remoteIP:  "LocalSubnet",
		},
		{
			name:      ruleGameplay,
			direction: "in",
			protocol:  "TCP",
			localPort: lanGameplayPort,
			program:   executablePath,
			remoteIP:  "LocalSubnet",
		},
		{
			name:      ruleDHT,
			direction: "in",
			protocol:  "UDP",
			localPort: lanGameplayPort,
			program:   executablePath,
			remoteIP:  "any",
		},
		{
			name:       ruleCloudflareTCP,
			direction:  "out",
			protocol:   "TCP",
			remotePort: cloudflareTunnelPort,
			program:    cloudflaredPath,
			remoteIP:   "any",
		},
		{
			name:       ruleCloudflareUDP,
			direction:  "out",
			protocol:   "UDP",
			remotePort: cloudflareTunnelPort,
			program:    cloudflaredPath,
			remoteIP:   "any",
		},
		// The health probe is made by GunMayhem, not cloudflared. Keep its
		// HTTPS access scoped to the current executable as well.
		{
			name:       ruleCloudflareHTTP,
			direction:  "out",
			protocol:   "TCP",
			remotePort: cloudflareHealthPort,
			program:    executablePath,
			remoteIP:   "any",
		},
	}
}

func (rule firewallRule) addArgs() []string {
	args := []string{
		"name=" + rule.name,
		"dir=" + rule.direction,
		"action=allow",
		"protocol=" + rule.protocol,
	}
	if rule.localPort > 0 {
		args = append(args, "localport="+strconv.Itoa(rule.localPort))
	}
	if rule.remotePort > 0 {
		args = append(args, "remoteport="+strconv.Itoa(rule.remotePort))
	}
	if rule.program != "" {
		args = append(args, "program="+rule.program)
	}
	if rule.remoteIP != "" {
		args = append(args, "remoteip="+rule.remoteIP)
	}
	return append(args, "profile=any", "enable=yes")
}

func firewallAddArgs(rule firewallRule) []string {
	return append([]string{"advfirewall", "firewall", "add", "rule"}, rule.addArgs()...)
}

func firewallDeleteArgs(name string) []string {
	return []string{"advfirewall", "firewall", "delete", "rule", "name=" + name}
}

func firewallShowArgs(name string) []string {
	return []string{"advfirewall", "firewall", "show", "rule", "name=" + name, "verbose"}
}

// firewallOutputHasProgram is deliberately path-aware: a rule with the right
// name but a program path from an older package must not suppress elevation.
func firewallOutputHasProgram(output []byte, program string) bool {
	program = firewallPathKey(program)
	if program == "" {
		return true
	}

	// Compare the complete value after any netsh field label. This works for
	// localized labels and avoids accepting an old path that is only a prefix
	// of the current executable path.
	text := strings.ReplaceAll(string(output), "\x00", "")
	for _, line := range strings.Split(text, "\n") {
		separator := strings.IndexByte(line, ':')
		if separator >= 0 && firewallPathKey(line[separator+1:]) == program {
			return true
		}
	}
	return false
}

func firewallPathKey(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\"")
	value = strings.ReplaceAll(value, `\`, "/")
	value = strings.TrimPrefix(value, "//?/")
	return strings.ToLower(value)
}
