package pv

import (
	"fmt"

	"k8s.io/api/core/v1"
	
	"github.com/fatih/color"
)

func HandlerCheck(act *ActionConfig) error {
	fmt.Printf("[%d] 进行资源安全检查\n", act.CurrentStep)

	var err error

	// check target node
	fmt.Printf("    检查目标节点 ")
	err = nodeCheck(act.TargetNode)
	if err != nil {
		color.Red("失败")
		return err
	}
	color.Green("正常")

	// check source node
	fmt.Printf("    检查源节点 ")
	err = nodeCheck(act.SourceNode)
	if err != nil {
		color.Red("失败")
		return err
	}
	color.Green("正常")

	// check if target and source are same one
	fmt.Printf("    检查是否为同一个节点  ")
	if act.TargetNode.ObjectMeta.Name == act.SourceNode.ObjectMeta.Name {
		color.Red("失败")
		return fmt.Errorf("source and distination are same node")
	}
	color.Green("正常")

	// var pvcp = act.PvcPairs[index]

	// check pv
	// var pvs = pvcp.pv.Status.Phase
	// fmt.Printf("    检查PV状态是否为有效绑定 ")
	// if pvs != v1.VolumeBound {
	// 	color.Red("失败")
	// 	return fmt.Errorf("we except pv %s status be Bound, but we got %s", pvcp.pv.ObjectMeta.Name, pvs)
	// }
	// color.Green("正常")

	// // check pvc
	// var pvcs = pvcp.pvc.Status.Phase
	// fmt.Printf("    检查PVC状态是否为有效绑定 ")
	// if pvcs != v1.ClaimBound {
	// 	color.Red("失败")
	// 	return fmt.Errorf("we expect pvc %s status be Bound, but we got %s", pvcp.pvc.ObjectMeta.Name, pvs)
	// }
	// color.Green("正常")

	// check pod
	// var ps = act.Pod.Status.Phase
	// fmt.Printf("    检查Pod状态是否正常 ")
	// if ps != v1.PodRunning && ps != v1.PodSucceeded {
	// 	color.Red("失败")
	// 	return fmt.Errorf("we expect node %s status be Running or Succeeded, bu we got %s", act.Pod.ObjectMeta.Name, ps)
	// }
	// color.Green("正常")

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