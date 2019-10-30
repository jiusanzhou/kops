package pv

// Config present all configuration
type Config struct {
	Source string `opts:"short=s,help=transport pvs from source node"` // source node
	Target string `opts:"short=t,help=transport pvs to target node"` // target node
	
	Username string `opts:"short=u,help=username for logon node"`
	Password string `opts:"short=p,help=password for login node"`

	Yes    bool   `opts:"short=y,help=no need to wait user's typo confirm"`
}

// NewConfig returns a new config
func NewConfig() *Config {
	return &Config{
		// add default value
	}
}