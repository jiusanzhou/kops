package pv

import (
	"errors"

	"k8s.io/api/core/v1"
)

var (
	ErrCancel      = errors.New("cancel")
	PatchPVRecycle = []byte(`{"spec": {"persistentVolumeReclaimPolicy": "Retain"}}`)
	PVNameSuffix   = "pvsynced" // special
)

type PVItem struct {
	OldPv, NewPv       *v1.PersistentVolume
	OldPvc, NewPvc      *v1.PersistentVolumeClaim
	SourcePath string
	TargetPath string
	FirstSynced bool
	Synced bool
	OldPvcDeleted bool
	PvCreated bool
	PvcCreated bool

	rsynccmd string
}

// ActionConfig presents how to transport a pv from a node to another
type ActionConfig struct {
	Pod        v1.Pod
	Items     []*PVItem
	SourceNode v1.Node
	TargetNode v1.Node
	CurrentStep int
	SrcHost  string
	DistHost string

	PodDeleted bool

	Finished bool
	Error string

	RollBack bool
	Continue bool

	m *Manager
}

type ActionHandler func(*ActionConfig) error

func (act *ActionConfig) next() error {
	var fn = act.m.fns[act.CurrentStep]
	if fn == nil {
		return errors.New("状态机错误: 未找到对应处理器")
	}
	var err = fn(act)
	if err != nil {
		return err
	}
	act.CurrentStep += 1
	if act.CurrentStep == len(act.m.fns) {
		return nil
	}
	return act.next()

}

// Run start to process
func (act *ActionConfig) Run() error {
	// prepre for configuration
	// recovery from log
	return act.next()
}

