package pv

import (
	"fmt"
	"github.com/pkg/errors"

	"k8s.io/api/core/v1"
)

// ActionConfig presents how to transport a pv from a node to another
type ActionConfig struct {
	Pod                   v1.Pod
	PersistentVolume      v1.PersistentVolume
	PersistentVolumeClaim v1.PersistentVolumeClaim
	SourceNode            v1.Node
	TargetNode            v1.Node

	progress int // total 100

	srcHost  string
	distHost string

	srcDataPath  string
	distDataPath string
}

// Run start to process
func (act *ActionConfig) Run() error {
	var err error

	// 0. check status. check all of nodes(source and distination), pv, pvc and pod.
	err = act.check()
	if err != nil {
		return err
	}

	// 1. sync data. start the rsync to sync data from source node to distinate node.
	err = act.syncdata()
	if err != nil {
		return err
	}

	// 2. delete pvc.
	//    **make sure data sync was completed, protect pod can't be deleted then, the
	//    **pvc will turn to terminaing status.
	err = act.deletepvc()
	if err != nil {
		return err
	}

	// 3. delete pod. pvc(pv) will be delete, if we need to keep data safe we need to
	//    change the pv's policy to retain or recycle before delete the pod.
	err = act.deletepod()
	if err != nil {
		return err
	}

	// 4. *sync data. sync data again, optional.*
	err = act.syncdata()
	if err != nil {
		return err
	}

	// 5. rename(or delete) pv (and data).
	err = act.renamepv()
	if err != nil {
		return err
	}

	// 5. use the original pv name to create a new pv on the new node with synced
	//    data (with path).
	err = act.createpv()
	if err != nil {
		return err
	}

	// 6. restore the pvc (actually reuse the name and pv refrence of pv).
	err = act.restorepvc()
	if err != nil {
		return err
	}

	// 7. waiting for pod scheduled.
	err = act.waitpodready()
	if err != nil {
		return err
	}

	return nil
}

func (act *ActionConfig) waitpodready() error {
	// timeout. 
	// TODO: we need to roll back all actions?

	// actually pvc status is BOUND
	// pvcclient.

	return nil
}

func (act *ActionConfig) restorepvc() error {

	return nil
}

func (act *ActionConfig) createpv() error {

	return nil
}

func (act *ActionConfig) renamepv() error {

	return nil
}

func (act *ActionConfig) deletepod() error {
	// **make sure pvc status to terminating!important

	return nil
}

func (act *ActionConfig) deletepvc() error {
	// modify recycle of pv?
	// **make sure data has been synced!important. rdiff?

	return nil
}

func (act *ActionConfig) syncdata() error {
	// start sync service to sync data
	// **make sure pv not be deleted

	return nil
}

func (act *ActionConfig) check() error {
	var err error

	// check target node
	err = nodeCheck(act.TargetNode)
	if err != nil {
		return errors.Wrap(err, "target node: ")
	}

	// check source node
	err = nodeCheck(act.SourceNode)
	if err != nil {
		return errors.Wrap(err, "source node")
	}

	// check if target and source are same one
	if act.TargetNode.ObjectMeta.Name == act.SourceNode.ObjectMeta.Name {
		return fmt.Errorf("source and distination are same node")
	}

	// check pv
	var pvs = act.PersistentVolume.Status.Phase
	if pvs != v1.VolumeBound {
		return fmt.Errorf("we except pv %s status be Bound, but we got %s",  act.PersistentVolume.ObjectMeta.Name, pvs)
	}

	// check pvc
	var pvcs = act.PersistentVolumeClaim.Status.Phase
	if pvcs != v1.ClaimBound {
		return fmt.Errorf("we expect pvc %s status be Bound, but we got %s",  act.PersistentVolumeClaim.ObjectMeta.Name, pvs)
	}

	// check pod
	var ps = act.Pod.Status.Phase
	if ps != v1.PodRunning && ps != v1.PodSucceeded {
		return fmt.Errorf("we expect node %s status be Running or Succeeded, bu we got %s", act.Pod.ObjectMeta.Name, ps)
	}

	// everything can be ok

	return nil
}

// nodeCheck
func nodeCheck(node v1.Node) error {
	_nodeTypes := []v1.NodeConditionType{} 
	for _, c := range node.Status.Conditions {
		if c.Status == v1.ConditionTrue {
			_nodeTypes = append(_nodeTypes, c.Type)
		}
	}

	// TODO: length can be 0?
	if len(_nodeTypes) != 1 || _nodeTypes[0] != v1.NodeReady {
		var msg = ""
		for _, t := range _nodeTypes {
			msg += string(t) + "; "
		}
		return fmt.Errorf("node %s check error%s", node.ObjectMeta.Name, msg)
	}

	return nil
}