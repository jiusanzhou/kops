package mv

// Manager to control mv
type Manager struct {
	Config *Config
}

func NewManager() *Manager {
	return &Manager{
		Config: NewConfig(),
	}
}