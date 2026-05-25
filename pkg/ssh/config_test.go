package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseSSHConfig_Basic(t *testing.T) {
	path := writeTempConfig(t, `
Host gateway
  HostName gateway.tailnet.example.com
  User ubuntu

Host mini
  HostName 192.168.1.10
  User mini
  Port 22

Host work-server
  HostName work.example.com
  User deploy
  Port 2222
  IdentityFile ~/.ssh/work_rsa
`)
	hosts, err := ParseSSHConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 3 {
		t.Fatalf("expected 3 hosts, got %d: %v", len(hosts), hosts)
	}

	// gateway
	if hosts[0].Name != "gateway" {
		t.Errorf("expected gateway, got %s", hosts[0].Name)
	}
	if hosts[0].HostName != "gateway.tailnet.example.com" {
		t.Errorf("wrong hostname: %s", hosts[0].HostName)
	}
	if hosts[0].Port != "22" {
		t.Errorf("expected default port 22, got %s", hosts[0].Port)
	}

	// work-server
	if hosts[2].Port != "2222" {
		t.Errorf("expected port 2222, got %s", hosts[2].Port)
	}
	if !strings.Contains(hosts[2].IdentityFile, "work_rsa") {
		t.Errorf("expected work_rsa in identity file, got %s", hosts[2].IdentityFile)
	}
}

func TestParseSSHConfig_SkipsWildcards(t *testing.T) {
	path := writeTempConfig(t, `
Host *
  ServerAliveInterval 30
  ServerAliveCountMax 3

Host gateway
  HostName gateway.example.com
  User ubuntu
`)
	hosts, err := ParseSSHConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host (wildcard skipped), got %d: %v", len(hosts), hosts)
	}
	if hosts[0].Name != "gateway" {
		t.Errorf("expected gateway, got %s", hosts[0].Name)
	}
}

func TestParseSSHConfig_Empty(t *testing.T) {
	path := writeTempConfig(t, "")
	hosts, err := ParseSSHConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(hosts))
	}
}

func TestParseSSHConfig_NotFound(t *testing.T) {
	hosts, err := ParseSSHConfig("/nonexistent/path/config")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(hosts))
	}
}

func TestParseSSHConfig_MultiHostLine(t *testing.T) {
	path := writeTempConfig(t, `
Host dev1 dev2 dev3
  HostName dev.example.com
  User dev
`)
	hosts, err := ParseSSHConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Multi-host lines: we only take the first name per spec
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host entry (first of multi), got %d", len(hosts))
	}
	if hosts[0].Name != "dev1" {
		t.Errorf("expected dev1, got %s", hosts[0].Name)
	}
}

func TestParseSSHConfig_Comments(t *testing.T) {
	path := writeTempConfig(t, `
# This is a comment
Host gateway
  # Another comment
  HostName gateway.example.com
  User ubuntu
`)
	hosts, err := ParseSSHConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
}

func TestParseSSHConfig_DefaultPath(t *testing.T) {
	// Passing empty path falls back to ~/.ssh/config — just verify no panic/error
	// (may or may not exist on this machine; both outcomes are valid)
	_, err := ParseSSHConfig("")
	if err != nil {
		t.Fatalf("unexpected error with default path: %v", err)
	}
}

func TestParseSSHConfig_IdentityFileAbsolute(t *testing.T) {
	path := writeTempConfig(t, `
Host myserver
  HostName myserver.example.com
  User admin
  IdentityFile /home/user/.ssh/id_rsa
`)
	hosts, err := ParseSSHConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].IdentityFile != "/home/user/.ssh/id_rsa" {
		t.Errorf("expected absolute identity file path, got %s", hosts[0].IdentityFile)
	}
}

func TestParseSSHConfig_Include(t *testing.T) {
	// Create an included config file
	dir := t.TempDir()

	includedContent := `Host included-host
  HostName included.example.com
  User included
`
	includedPath := filepath.Join(dir, "included_config")
	if err := os.WriteFile(includedPath, []byte(includedContent), 0600); err != nil {
		t.Fatal(err)
	}

	mainContent := "Include " + includedPath + `

Host main-host
  HostName main.example.com
  User main
`
	mainPath := filepath.Join(dir, "config")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0600); err != nil {
		t.Fatal(err)
	}

	hosts, err := ParseSSHConfig(mainPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts (1 included + 1 main), got %d: %v", len(hosts), hosts)
	}
	if hosts[0].Name != "included-host" {
		t.Errorf("expected included-host first, got %s", hosts[0].Name)
	}
	if hosts[1].Name != "main-host" {
		t.Errorf("expected main-host second, got %s", hosts[1].Name)
	}
}

func TestParseSSHConfig_IncludeLoop(t *testing.T) {
	// A file that includes itself should not loop forever
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "config")

	// Write a self-referencing config
	content := "Include " + mainPath + `

Host gateway
  HostName gateway.example.com
`
	if err := os.WriteFile(mainPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	hosts, err := ParseSSHConfig(mainPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should still parse gateway despite loop protection
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
}

func TestIsWildcard(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"*", true},
		{"*.example.com", true},
		{"dev?", true},
		{"gateway", false},
		{"mini", false},
	}
	for _, tc := range cases {
		if got := isWildcard(tc.name); got != tc.want {
			t.Errorf("isWildcard(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}
