package pv

import (
	"fmt"
)

// Manager controll things
type Manager struct {
	Config *Config
}

// NewManager create a new manager
func NewManager() *Manager {
	return &Manager{
		Config: NewConfig(),
	}
}

// Start process
func (m *Manager) Start(pods ...string) {
	fmt.Println("pods:", pods)
}