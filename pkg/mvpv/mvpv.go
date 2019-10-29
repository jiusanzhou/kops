package mvpv

import (
	"fmt"
	"strings"
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
	
	// 2. delete pvc.
	//    **make sure data sync was completed, protect pod can't be deleted then, the
	//    pvc will turn to terminaing status.

	// 3. delete pod. pvc(pv) will be delete, if we need to keep data safe we need to
	//    change the pv's policy to retain or recycle before delete the pod.

	// 4. *sync data. sync data again, optional.*

	// 5. rename(or delete) pv (and data).

	// 5. use the original pv name to create a new pv on the new node with synced
	//    data (with path).

	// 6. restore the pvc (actually reuse the name and pv refrence of pv).

	// 7. waiting for pod scheduled.

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

	// check pv
	var pvs = act.PersistentVolume.Status.PersistentVolumeStatus.Phase
	if pvs != v1.VolumeBound {
		return errors.New("we except pv %s status be Bound, but we got %s",  act.PersistentVolume.ObjectMeta.Name, pvs)
	}

	// check pvc
	var pvcs = act.PersistentVolumeClaim.Status.PersistentVolumeClaimPhase.Phase
	if pvcs != v1.ClaimBound {
		return errors.New("we expect pvc %s status be Bound, but we got %s",  act.PersistentVolumeClaim.ObjectMeta.Name, pvs)
	}

	// check pod
	var ps = act.Pod.Status.Phase
	if ps != v1.PodRunning && ps != v1.PodSucceeded {
		return errors.New("we expect node %s status be Running or Succeeded, bu we got %s", act.Pod.ObjectMeta.Name, ps)
	}

	// everything can be ok

	return nil
}

// nodeCheck
func nodeCheck(node v1.Node) error {
	_nodeTypes := []string{} 
	for c := range act.TargetNode.Status.Conditions {
		if c.Status == "True" {
			_nodeTypes = append(_nodeTypes, c.Type)
		}
	}

	if len(_nodeTypes) != 1 || _nodeTypes[0] != v1.NodeReady {
		return fmt.Errorf("node %s check error%s", node.ObjectMeta.Name, strings.Join(_nodeTypes, "; "))
	}

	return nil
}