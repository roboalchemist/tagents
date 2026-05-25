package ssh

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ParseSSHConfig parses ~/.ssh/config and returns all non-wildcard Host entries.
// It handles Include directives recursively.
func ParseSSHConfig(configPath string) ([]Host, error) {
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configPath = filepath.Join(home, ".ssh", "config")
	}
	return parseFile(configPath, make(map[string]bool))
}

func parseFile(path string, visited map[string]bool) ([]Host, error) {
	if visited[path] {
		return nil, nil // avoid include loops
	}
	visited[path] = true

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var hosts []Host
	var current *Host
	baseDir := filepath.Dir(path)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "include":
			// Expand glob patterns relative to ssh config dir
			pattern := value
			if !filepath.IsAbs(pattern) {
				pattern = filepath.Join(baseDir, pattern)
			}
			matches, err := filepath.Glob(pattern)
			if err == nil {
				for _, match := range matches {
					included, err := parseFile(match, visited)
					if err == nil {
						hosts = append(hosts, included...)
					}
				}
			}
		case "host":
			// Save previous host
			if current != nil && !isWildcard(current.Name) {
				hosts = append(hosts, *current)
			}
			// Start new host (may be space-separated multi-host — take first)
			names := strings.Fields(value)
			if len(names) > 0 {
				current = &Host{Name: names[0], Port: "22"}
			} else {
				current = nil
			}
		case "hostname":
			if current != nil {
				current.HostName = value
			}
		case "user":
			if current != nil {
				current.User = value
			}
		case "port":
			if current != nil {
				current.Port = value
			}
		case "identityfile":
			if current != nil {
				if !filepath.IsAbs(value) && strings.HasPrefix(value, "~") {
					home, _ := os.UserHomeDir()
					value = filepath.Join(home, value[2:])
				}
				current.IdentityFile = value
			}
		}
	}
	// Don't forget the last host
	if current != nil && !isWildcard(current.Name) {
		hosts = append(hosts, *current)
	}
	return hosts, scanner.Err()
}

func isWildcard(name string) bool {
	return strings.ContainsAny(name, "*?")
}
