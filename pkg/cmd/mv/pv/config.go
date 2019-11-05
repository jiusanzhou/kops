package pv

// Config present all configuration
type Config struct {
	Source string `opts:"short=s,help=transport pvs from source node"` // source node
	Target string `opts:"short=t,help=transport pvs to target node"`   // target node

	Username string `opts:"short=u,help=username for logon node"`
	Password string `opts:"short=p,help=password for login node"`

	DryRun bool `opts:"help=dry run command no modify"`

	Yes bool `opts:"short=y,help=no need to wait user's typo confirm"`

	Namespace string `opts:"short=n,help=namepace to process"`

	AutoCreate bool `opts:"help=auto create direcotry if not exits"`

	ForceWrite bool `opts:"help=if target exits create a new path"`

	DaemonRsync bool `opts:"short=d,help=use rsync daemon to sync data"`

	RsyncArgs string `opts:help=args and flags for rsync`
}

// NewConfig returns a new config
func NewConfig() *Config {
	return &Config{
		// add default value
		Namespace: "default",
		RsyncArgs: "-az",
	}
}
