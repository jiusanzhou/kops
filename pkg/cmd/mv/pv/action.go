package pv

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.zoe.im/x/sh"
	
	"go.zoe.im/kops/pkg/utils"
)

var (
	ErrCancel = errors.New("cancel")
	PatchPVRecycle = []byte(`{"spec": {"persistentVolumeReclaimPolicy": "Retain"}}`)
	PVNameSuffix = "-pvsynced"
)

type PVCPair struct {
	pv  v1.PersistentVolume
	pvc v1.PersistentVolumeClaim
}

// ActionConfig presents how to transport a pv from a node to another
type ActionConfig struct {
	Pod                   v1.Pod
	PvcPairs              []PVCPair
	SourceNode            v1.Node
	TargetNode            v1.Node

	progress int // total 100

	srcHost  string
	distHost string

	m *Manager
}

// Run start to process
func (act *ActionConfig) Run() error {
	var err error

	for index, _ := range act.PvcPairs {

		// 0. check status. check all of nodes(source and distination), pv, pvc and pod.
		err = act.check(index)
		if err != nil {
			return err
		}
	
		// 1. sync data. start the rsync to sync data from source node to distinate node.
		err = act.syncdata(index)
		if err != nil {
			return err
		}
	
		// 2. delete pvc.
		//    **make sure data sync was completed, protect pod can't be deleted then, the
		//    **pvc will turn to terminaing status.
		err = act.deletepvc(index)
		if err != nil {
			return err
		}
	
		// 3. delete pod. pvc(pv) will be delete, if we need to keep data safe we need to
		//    change the pv's policy to retain or recycle before delete the pod.
		err = act.deletepod(index)
		if err != nil {
			return err
		}

		// waiting for pvc deleted
		act.waitingpvcdeleted(index)
	
		// 4. *sync data. sync data again, optional.*
		// err = act.syncdata()
		// if err != nil {
		// 	return err
		// }
	
		// 5. rename(or delete) pv (and data).
		err = act.renamepv(index)
		if err != nil {
			return err
		}
	
		// 5. use the original pv name to create a new pv on the new node with synced
		//    data (with path).
		err = act.createpv(index)
		if err != nil {
			return err
		}
	
		// 6. restore the pvc (actually reuse the name and pv refrence of pv).
		err = act.restorepvc(index)
		if err != nil {
			return err
		}
	
		// 7. waiting for pod scheduled.
		err = act.waitpodready(index)
		if err != nil {
			return err
		}
	}

	return nil
}

func (act *ActionConfig) waitpodready(index int) error {
	// timeout. 
	// TODO: we need to roll back all actions?

	// actually pvc status is BOUND
	// pvcclient.
	fmt.Println("[INFO] [7] 重启Pod", act.Pod.Name)

	_ = podclient.Delete(act.Pod.Name, nil)
	// if err != nil {
	// 	fmt.Println("[ERROR] [7] 重启Pod", act.Pod.Name, "失败:", err)
	// 	return err
	// }
	return nil
}

func (act *ActionConfig) restorepvc(index int) error {
	var opvc = act.PvcPairs[index].pvc
	var pv = act.PvcPairs[index].pv
	fmt.Println("[INFO] [6] 恢复创建PVC", opvc.Name)
	var npvc = v1.PersistentVolumeClaim{
		TypeMeta: opvc.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name: opvc.ObjectMeta.Name,
			Namespace: opvc.ObjectMeta.Namespace,
			Labels: opvc.ObjectMeta.Labels,
			Annotations: opvc.ObjectMeta.Annotations,
			ClusterName: opvc.ObjectMeta.ClusterName,
			ManagedFields: opvc.ObjectMeta.ManagedFields,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: opvc.Spec.AccessModes,
			Selector: opvc.Spec.Selector,
			Resources: opvc.Spec.Resources,
			VolumeName: pv.Name, // important
			StorageClassName: opvc.Spec.StorageClassName,
			VolumeMode: opvc.Spec.VolumeMode,
			DataSource: opvc.Spec.DataSource,
		},
	}
	_, err := pvcclient.Create(&npvc)
	if err != nil {
		fmt.Println("[INFO] [6] 恢复创建PVC", npvc.Name, "失败:", err)
		return err
	}
	fmt.Println("[INFO] [6] 恢复创建PVC", npvc.Name, "成功")
	return nil
}

func (act *ActionConfig) createpv(index int) error {
	// 1. rename from the old one
	// 2. change the node affinity to the new node
	// 3. change the data path if need
	fmt.Println("[INFO] [5] 在新节点上创建PV")
	var opv = act.PvcPairs[index].pv
	var npv = v1.PersistentVolume{
		TypeMeta: opv.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name: opv.ObjectMeta.Name + PVNameSuffix,
			Namespace: opv.ObjectMeta.Namespace,
			Labels: opv.ObjectMeta.Labels,
			Annotations: opv.ObjectMeta.Annotations,
			ClusterName: opv.ObjectMeta.ClusterName,
			ManagedFields: opv.ObjectMeta.ManagedFields,
		},
		Spec: v1.PersistentVolumeSpec{
			AccessModes: opv.Spec.AccessModes,
			Capacity: opv.Spec.Capacity,
			PersistentVolumeSource: opv.Spec.PersistentVolumeSource,
			PersistentVolumeReclaimPolicy: opv.Spec.PersistentVolumeReclaimPolicy,
			StorageClassName: opv.Spec.StorageClassName,
			MountOptions: opv.Spec.MountOptions,
			VolumeMode: opv.Spec.VolumeMode,
			NodeAffinity: opv.Spec.NodeAffinity,
		},
	}

	// TODO: just panic if empty???
	for i, term := range npv.Spec.NodeAffinity.Required.NodeSelectorTerms {
		for x, me := range term.MatchExpressions {
			if me.Operator == v1.NodeSelectorOpIn {
				me.Values = []string{act.TargetNode.ObjectMeta.Name} // new node
			}
			npv.Spec.NodeAffinity.Required.NodeSelectorTerms[i].MatchExpressions[x] = me
		}
		for x, mf := range  term.MatchFields {
			if mf.Operator == v1.NodeSelectorOpIn {
				mf.Values = []string{act.TargetNode.ObjectMeta.Name} // new node
			}
			npv.Spec.NodeAffinity.Required.NodeSelectorTerms[i].MatchExpressions[x] = mf
		}
	}
	act.PvcPairs[index].pv = npv

	_, err := pvclient.Create(&npv)
	if err != nil {
		fmt.Println("[ERROR] [5] 创建PV", npv.Name, "失败:", err)
		return err
	}
	fmt.Println("[INFO] [5] 创建PV", npv.Name, "成功")
	return nil
}

func (act *ActionConfig) renamepv(index int) error {
	// TODO:
	fmt.Println("[WARN] [4] 不删除PV")
	return nil
}

func (act *ActionConfig) deletepod(index int) error {
	// **make sure pvc status to terminating!important
	fmt.Println("[INFO] [3] 删除Pod", act.Pod.Name)
	return podclient.Delete(act.Pod.Name, nil)
}

func (act *ActionConfig) waitingpvcdeleted(index int) {
	fmt.Printf("等待")
	for {
		fmt.Printf(".")
		time.Sleep(time.Second)
		_, err := pvcclient.Get(act.PvcPairs[index].pvc.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Println()
			return
		}
	}
}

func (act *ActionConfig) deletepvc(index int) error {
	// modify recycle of pv?
	// **make sure data has been synced!important. rdiff?

	var pvp = act.PvcPairs[index]
	_, err := pvclient.Patch(pvp.pv.Name, types.MergePatchType, PatchPVRecycle)
	fmt.Println("[INFO] [2] 调整PV", pvp.pv.Name, "的回收策略为 Retain")
	if err != nil {
		fmt.Println("[ERROR] [2] 调整PV回收策略失败", err)
		return err
	}
	fmt.Println("[INFO] [2] 删除PVC", pvp.pvc.Name)
	var c int64 = 0
	return pvcclient.Delete(pvp.pvc.Name, &metav1.DeleteOptions{GracePeriodSeconds: &c})
}

func (act *ActionConfig) syncdata(index int) error {
	// start sync service to sync data
	// **make sure pv not be deleted
	fmt.Println("[INFO] [1] 同步数据")
	var m = act.m

	var err error

	var pvp = act.PvcPairs[index]

	fmt.Println("[INFO] [1] 同步第", index+1, "个PV数据")

	_path := pvp.pv.Spec.Local.Path
	_parent := utils.ParentPath(_path)
	
	_sourcepath := _path
	_targetpath := _parent

	// 1. check if target exits
	if utils.Exits(_path) {
		fmt.Println("[ERROR] 路径", _path, "已存在")
		if !m.Config.ForceWrite {
			if !m.Config.AutoCreate {
				return ErrCancel
			} else {
				// generate a new path
				_sourcepath = _path + "/*"
				_targetpath = _path + "-moved-" + pvp.pv.ObjectMeta.ResourceVersion
				fmt.Println("[INFO] 自动生成新路径", _targetpath)
			}
		} else {
			fmt.Println("[WARN] 即将强制覆写路径", _path)
		}
	}

	// 2. create parent directory of target pv
	if !utils.Exits(_parent) {
		fmt.Println("上一级目录", _parent, "不存在")
		if !m.Config.AutoCreate {
			return ErrCancel
		}
		fmt.Println("[WARN] 自动创建父级路径", _parent)
		err = sh.Run("mkdir -r " + _parent)
		if err != nil {
			fmt.Println("[ERROR] 创建目录", _parent, "失败:", err)
		}
	}
	if _sourcepath[len(_sourcepath) - 1] == '*' {
		// special directory we need to make sure target exits
		if !utils.Exits(_targetpath) {
			fmt.Println("[WARN] 目标目录", _targetpath, "不存在, 自动创建")
			err = sh.Run("mkdir " + _targetpath)
			if err != nil {
				fmt.Println("[ERROR] 创建目录", _targetpath, "失败:", err)
			}
		}
	}

	// 3. sync data use rsync
	var data = map[string]string{
		"args": m.Config.RsyncArgs,
		"source_host": act.srcHost,
		"source_path": _sourcepath,
		"target_path": _targetpath,
	}
	if len(m.Config.Username) > 0 {
		data["username"] = m.Config.Username
	}

	rsynccmd := genRsyncCmd(m.Config.DaemonRsync, data)

	if m.Config.DryRun {
		fmt.Println("[DEBUG] 运行命令", rsynccmd)
		return nil
	}

	err = sh.Run(rsynccmd)
	if err != nil {
		return err
	}

	return nil
}

func (act *ActionConfig) check(index int) error {
	fmt.Println("[INFO] [0] 检测资源状态")

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

	var pvcp = act.PvcPairs[index]


		// check pv
		var pvs = pvcp.pv.Status.Phase
		if pvs != v1.VolumeBound {
			return fmt.Errorf("we except pv %s status be Bound, but we got %s",  pvcp.pv.ObjectMeta.Name, pvs)
		}
	
		// check pvc
		var pvcs = pvcp.pvc.Status.Phase
		if pvcs != v1.ClaimBound {
			return fmt.Errorf("we expect pvc %s status be Bound, but we got %s",  pvcp.pvc.ObjectMeta.Name, pvs)
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

// imporant!!!
func genRsyncCmd(daemon bool, data map[string]string) string {
	var cmd = "rsync"
	cmd += " " + data["args"]
	if username, ok := data["username"]; ok {
		cmd += " " + username + "@" + data["source_host"]
	} else {
		cmd += " " + data["source_host"]
	}
	cmd += ":"
	if daemon {
		cmd += ":"
	}
	cmd += data["source_path"]
	cmd += " " + data["target_path"]
	return cmd
}