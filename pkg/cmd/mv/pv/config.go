package pv

// Config present all configuration
type Config struct {
	Source      string `opts:"short=s,help=transport pvs from source node"` // source node
	Target      string `opts:"short=t,help=transport pvs to target node"`   // target node
	Username    string `opts:"short=u,help=username for logon node"`
	Password    string `opts:"short=p,help=password for login node"`
	DryRun      bool   `opts:"help=dry run command no modify"`
	Yes         bool   `opts:"short=y,help=no need to wait user's typo confirm"`
	Namespace   string `opts:"short=n,help=namepace to process"`
	AutoCreate  bool   `opts:"short=a,help=auto create direcotry if not exits"`
	ForceWrite  bool   `opts:"help=if target exits create a new path"`
	DaemonRsync bool   `opts:"help=use rsync daemon to sync data"`
	RsyncArgs   string `opts:"help=args and flags for rsync"`
	Directory   string `opts:"short=d,help=target directory of the pv data direcotry store; if empty use orignal path"`
	Prefix      string `opts:"help=prefix of pv directory and name"`
	Wait        int    `opts:"short=w,help=timeout for waitting pvc deleted; 0 for disable; seconds"`
	LogFile     string `opts:"short=f,help=last run exitting log; try to revcovery"`
	Step        int    `opts:"help=set current step"`
}

// NewConfig returns a new config
func NewConfig() *Config {
	return &Config{
		// add default value
		Namespace: "default",
		RsyncArgs: "-az",
		Prefix:    "pvsynced",
		Wait:      0,
	}
}
