package session

import "github.com/roboalchemist/tagents/pkg/runtime"

type AgentSession struct {
	Machine string
	Name    string
	Runtime runtime.Runtime
	Status  runtime.Status
	CWD     string
	Preview string
}
