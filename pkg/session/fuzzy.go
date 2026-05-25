package session

import (
	"fmt"
	"strings"
)

// FuzzyMatch finds the best matching session for a query string.
//
// Query formats:
//   - "sim-1"         — fuzzy match against Name across all machines
//   - "vps1:sim-1"    — exact machine:name match
//   - "local:sim-1"   — match on local machine (machine == "")
//
// Returns error if no match or ambiguous (multiple matches).
func FuzzyMatch(sessions []AgentSession, query string) (*AgentSession, error) {
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no agent sessions found")
	}

	// Check for machine:name format
	if idx := strings.Index(query, ":"); idx != -1 {
		machine := query[:idx]
		name := query[idx+1:]
		return exactMatch(sessions, machine, name)
	}

	// Fuzzy match on name
	return fuzzyNameMatch(sessions, query)
}

// exactMatch finds a session with specific machine and name.
func exactMatch(sessions []AgentSession, machine, name string) (*AgentSession, error) {
	var matches []AgentSession
	for _, s := range sessions {
		sessionMachine := s.Machine
		if sessionMachine == "" {
			sessionMachine = "local"
		}
		if strings.EqualFold(sessionMachine, machine) && strings.EqualFold(s.Name, name) {
			matches = append(matches, s)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("agent %q not found on machine %q", name, machine)
	}
	return &matches[0], nil
}

// fuzzyNameMatch finds sessions whose name contains the query as a substring.
func fuzzyNameMatch(sessions []AgentSession, query string) (*AgentSession, error) {
	queryLower := strings.ToLower(query)

	// First: exact match on name
	var exact []AgentSession
	for _, s := range sessions {
		if strings.EqualFold(s.Name, query) {
			exact = append(exact, s)
		}
	}
	if len(exact) == 1 {
		return &exact[0], nil
	}
	if len(exact) > 1 {
		names := make([]string, len(exact))
		for i, s := range exact {
			names[i] = formatRef(s)
		}
		return nil, fmt.Errorf("ambiguous agent name %q, matches: %s", query, strings.Join(names, ", "))
	}

	// Second: substring match
	var partial []AgentSession
	for _, s := range sessions {
		if strings.Contains(strings.ToLower(s.Name), queryLower) {
			partial = append(partial, s)
		}
	}
	if len(partial) == 1 {
		return &partial[0], nil
	}
	if len(partial) > 1 {
		names := make([]string, len(partial))
		for i, s := range partial {
			names[i] = formatRef(s)
		}
		return nil, fmt.Errorf("ambiguous agent name %q, matches: %s", query, strings.Join(names, ", "))
	}

	return nil, fmt.Errorf("agent %q not found", query)
}

// formatRef returns a machine:name reference string.
func formatRef(s AgentSession) string {
	if s.Machine == "" {
		return s.Name
	}
	return s.Machine + ":" + s.Name
}
