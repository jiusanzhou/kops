package pv

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/fatih/color"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"go.zoe.im/x/sh"

	"go.zoe.im/kops/pkg/utils"
)

var (
	ErrCancel      = errors.New("cancel")
	PatchPVRecycle = []byte(`{"spec": {"persistentVolumeReclaimPolicy": "Retain"}}`)
	PVNameSuffix   = "pvsynced" // special
)

func withSuffixPVName(name string) string {
	var splits = strings.Split(name, "-")
	var l = len(splits)
	if l >= 3 && splits[l-2] == PVNameSuffix {
		// maybe this pv is a synced one, just replace the timestamp
		splits[l-1] = fmt.Sprintf("%d", time.Now().Unix())
		return strings.Join(splits, "-")
	} else {
		return fmt.Sprintf("%s-%s-%d", name, PVNameSuffix, time.Now().Unix())
	}
}

type PVCPair struct {
	pv       v1.PersistentVolume
	pvc      v1.PersistentVolumeClaim
	rsynccmd string
}

// ActionConfig presents how to transport a pv from a node to another
type ActionConfig struct {
	Pod        v1.Pod
	PvcPairs   []*PVCPair
	SourceNode v1.Node
	TargetNode v1.Node

	progress int // total 100

	srcHost  string
	distHost string

	m *Manager
}

// Run start to process
func (act *ActionConfig) Run() error {
	var err error

	PVNameSuffix = act.m.Config.Prefix

	for index := range act.PvcPairs {

		// 0. check status. check all of nodes(source and distination), pv, pvc and pod.
		err = act.check(index, 0)
		if err != nil {
			return err
		}

		// 1. sync data. start the rsync to sync data from source node to distinate node.
		err = act.syncdata(index, 1, true)
		if err != nil {
			return err
		}

		// 2. delete pvc.
		//    **make sure data sync was completed, protect pod can't be deleted then, the
		//    **pvc will turn to terminaing status.
		err = act.deletepvc(index, 2)
		if err != nil {
			return err
		}

		// 3. delete pod. pvc(pv) will be delete, if we need to keep data safe we need to
		//    change the pv's policy to retain or recycle before delete the pod.
		err = act.deletepod(index, 3)
		if err != nil {
			return err
		}

		// 4. *sync data. sync data again, optional.*
		err = act.syncdata(index, 4, false)
		if err != nil {
			return err
		}

		// 5. rename(or delete) pv (and data).
		err = act.renamepv(index, 5)
		if err != nil {
			return err
		}

		// 6. use the original pv name to create a new pv on the new node with synced
		//    data (with path).
		err = act.createpv(index, 6)
		if err != nil {
			return err
		}

		// 7. restore the pvc (actually reuse the name and pv refrence of pv).
		err = act.restorepvc(index, 7)
		if err != nil {
			return err
		}

		// 8. waiting for pod scheduled.
		err = act.waitpodready(index, 8)
		if err != nil {
			return err
		}
	}

	return nil
}

func (act *ActionConfig) waitpodready(index, step int) error {
	// timeout.
	// TODO: we need to roll back all actions?

	// actually pvc status is BOUND
	// pvcclient.
	fmt.Printf("[%d] 重启Pod %s ", step, act.Pod.Name)

	_ = podclient.Delete(act.Pod.Name, nil)
	// if err != nil {
	// 	fmt.Println("[7] 重启Pod", act.Pod.Name, "失败:", err)
	// 	return err
	// }
	color.Green("成功")
	return nil
}

func (act *ActionConfig) restorepvc(index, step int) error {
	var opvc = act.PvcPairs[index].pvc
	var pv = act.PvcPairs[index].pv
	fmt.Printf("[%d] 恢复创建PVC %s ", step, opvc.Name)
	var npvc = v1.PersistentVolumeClaim{
		TypeMeta: opvc.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:          opvc.ObjectMeta.Name,
			Namespace:     opvc.ObjectMeta.Namespace,
			Labels:        opvc.ObjectMeta.Labels,
			Annotations:   opvc.ObjectMeta.Annotations,
			ClusterName:   opvc.ObjectMeta.ClusterName,
			ManagedFields: opvc.ObjectMeta.ManagedFields,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes:      opvc.Spec.AccessModes,
			Selector:         opvc.Spec.Selector,
			Resources:        opvc.Spec.Resources,
			VolumeName:       pv.Name, // important
			StorageClassName: opvc.Spec.StorageClassName,
			VolumeMode:       opvc.Spec.VolumeMode,
			DataSource:       opvc.Spec.DataSource,
		},
	}
	_, err := pvcclient.Create(&npvc)
	if err != nil {
		color.Red("失败")
		fmt.Printf("    error: %s\n", err)
		return err
	}
	color.Green("成功")
	return nil
}

func (act *ActionConfig) createpv(index, step int) error {
	// 1. rename from the old one
	// 2. change the node affinity to the new node
	// 3. change the data path if need
	fmt.Printf("[%d] 在新节点上创建PV ", step)
	var opv = act.PvcPairs[index].pv
	var npv = v1.PersistentVolume{
		TypeMeta: opv.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:          withSuffixPVName(opv.ObjectMeta.Name),
			Namespace:     opv.ObjectMeta.Namespace,
			Labels:        opv.ObjectMeta.Labels,
			Annotations:   opv.ObjectMeta.Annotations,
			ClusterName:   opv.ObjectMeta.ClusterName,
			ManagedFields: opv.ObjectMeta.ManagedFields,
		},
		Spec: v1.PersistentVolumeSpec{
			AccessModes:                   opv.Spec.AccessModes,
			Capacity:                      opv.Spec.Capacity,
			PersistentVolumeSource:        opv.Spec.PersistentVolumeSource,
			PersistentVolumeReclaimPolicy: opv.Spec.PersistentVolumeReclaimPolicy,
			StorageClassName:              opv.Spec.StorageClassName,
			MountOptions:                  opv.Spec.MountOptions,
			VolumeMode:                    opv.Spec.VolumeMode,
			NodeAffinity:                  opv.Spec.NodeAffinity,
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
		for x, mf := range term.MatchFields {
			if mf.Operator == v1.NodeSelectorOpIn {
				mf.Values = []string{act.TargetNode.ObjectMeta.Name} // new node
			}
			npv.Spec.NodeAffinity.Required.NodeSelectorTerms[i].MatchExpressions[x] = mf
		}
	}
	act.PvcPairs[index].pv = npv

	_, err := pvclient.Create(&npv)
	if err != nil {
		color.Red("失败")
		fmt.Printf("    error: %s\n", err)
		return err
	}
	color.Green("成功")
	return nil
}

func (act *ActionConfig) renamepv(index, step int) error {
	// TODO:
	fmt.Printf("[%d] 重命名原PV ", step)
	color.Yellow("跳过")
	return nil
}

func (act *ActionConfig) deletepod(index, step int) error {
	// **make sure pvc status to terminating!important
	fmt.Printf("[%d] 删除Pod %s ", step, act.Pod.Name)
	err := podclient.Delete(act.Pod.Name, nil)
	if err != nil {
		color.Red("失败")
		return err
	}
	color.Green("成功")

	fmt.Printf("    等待PVC被删除 ")
	act.waitingpvcdeleted(index)
	color.Green(" 完成")
	return nil
}

func (act *ActionConfig) waitingpvcdeleted(index int) {
	// FIXME: use watch
	var c = 20
	for {
		fmt.Printf(".")
		time.Sleep(time.Second)
		_, err := pvcclient.Get(act.PvcPairs[index].pvc.Name, metav1.GetOptions{})
		if err != nil {
			return
		}
		c -= 1
		if c == 0 {
			return
		}
	}
}

func (act *ActionConfig) deletepvc(index, step int) error {
	// modify recycle of pv?
	// **make sure data has been synced!important. rdiff?
	fmt.Printf("[%d] 删除PVC\n", step)

	var pvp = act.PvcPairs[index]
	_, err := pvclient.Patch(pvp.pv.Name, types.MergePatchType, PatchPVRecycle)
	fmt.Printf("    调整PV %s 的回收策略为 Retain ", pvp.pv.Name)
	if err != nil {
		color.Red("失败")
		fmt.Printf("    error: %s\n", err)
		return err
	}

	// wait for recycle pf pv has been changed to retain
	// FIXME: use watch
	for {
		time.Sleep(time.Second)
		pv, err := pvclient.Get(act.PvcPairs[index].pv.Name, metav1.GetOptions{})
		if err == nil {
			if pv.Spec.PersistentVolumeReclaimPolicy == "Retain" {
				color.Green("成功")
				break
			}
		}
	}

	fmt.Printf("    删除PVC %s ", pvp.pvc.Name)
	var c int64 = 0
	err = pvcclient.Delete(pvp.pvc.Name, &metav1.DeleteOptions{GracePeriodSeconds: &c})
	if err != nil {
		color.Red("失败")
		return err
	}

	color.Green("成功")

	return nil
}

func (act *ActionConfig) syncdata(index, step int, created bool) error {
	// start sync service to sync data
	// **make sure pv not be deleted

	var (
		err         error
		_path       string
		_parent     string
		_sourcepath string
		_targetpath string

		data map[string]string

		m   = act.m
		pvp = act.PvcPairs[index]
	)

	if !created {
		fmt.Printf("[%d] 再次同步第 %d 个PV数据\n", step, index+1)
		goto runsync
	} else {
		fmt.Printf("[%d] 同步第 %d 个PV数据\n", step, index+1)
	}

	_path = pvp.pv.Spec.Local.Path
	_parent = utils.ParentPath(_path)

	if act.m.Config.Directory != "" {
		_parent = act.m.Config.Directory
	}

	_sourcepath = _path
	_targetpath = _parent

	// always use a new path name
	_sourcepath = _path + "/*"
	_targetpath = _parent + "/" + PVNameSuffix +"-" + pvp.pv.ObjectMeta.ResourceVersion


	// // 1. check if target exits
	// if utils.Exits(_path) {
	// 	fmt.Printf("[%d] 路径 %s 已存在\n", step, _path)
	// 	if !m.Config.ForceWrite {
	// 		if !m.Config.AutoCreate {
	// 			return ErrCancel
	// 		} else {
	// 			// generate a new path
	// 			_sourcepath = _path + "/*"
	// 			_targetpath = _path + "-moved-" + pvp.pv.ObjectMeta.ResourceVersion
	// 			fmt.Printf("[%d] 自动生成新路径 %s\n", step, _targetpath)
	// 		}
	// 	} else {
	// 		fmt.Printf("[%d] 即将强制覆写路径 %s\n", step, _path)
	// 	}
	// }

	// 2. create parent directory of target pv
	if !utils.Exits(_parent) {
		fmt.Printf("    上一级目录 %s 不存在 ", step, _parent)
		if !m.Config.AutoCreate {
			color.Red("失败")
			return ErrCancel
		}
		fmt.Printf("    自动创建父级路径 %s ", step, _parent)
		err = sh.Run("mkdir -p " + _parent)
		if err != nil {
			color.Red("失败")
			fmt.Printf("    error: %s\n", err)
			return err
		}
		color.Green("成功")
	}

	if _sourcepath[len(_sourcepath)-1] == '*' {
		// special directory we need to make sure target exits
		if !utils.Exits(_targetpath) {
			fmt.Printf("    目标目录 %s 不存在, 自动创建 ", _targetpath)
			err = sh.Run("mkdir " + _targetpath)
			if err != nil {
				color.Red("失败")
				fmt.Printf("    error: %s\n", err)
				return err
			}
			color.Green("成功")
		}
	}

	// 3. sync data use rsync
	data = map[string]string{
		"args":        m.Config.RsyncArgs,
		"source_host": act.srcHost,
		"source_path": _sourcepath,
		"target_path": _targetpath,
	}
	if len(m.Config.Username) > 0 {
		data["username"] = m.Config.Username
	}

	pvp.rsynccmd = genRsyncCmd(m.Config.DaemonRsync, data)

	if m.Config.DryRun {
		fmt.Println("    运行命令", pvp.rsynccmd)
		return nil
	}

runsync:
	fmt.Printf("    ")
	err = sh.Run(pvp.rsynccmd)
	fmt.Printf("    启动rsync同步数据 ")
	if err != nil {
		color.Red("失败")
		return err
	}
	color.Green("成功")

	return nil
}

func (act *ActionConfig) check(index, step int) error {
	fmt.Printf("[%d] 进行资源安全检查\n", step)

	var err error

	// check target node
	fmt.Printf("    检查目标节点 ")
	err = nodeCheck(act.TargetNode)
	if err != nil {
		color.Red("失败")
		return errors.Wrap(err, "target node: ")
	}
	color.Green("正常")

	// check source node
	fmt.Printf("    检查源节点 ")
	err = nodeCheck(act.SourceNode)
	if err != nil {
		color.Red("失败")
		return errors.Wrap(err, "source node")
	}
	color.Green("正常")

	// check if target and source are same one
	fmt.Printf("    检查是否为同一个节点  ")
	if act.TargetNode.ObjectMeta.Name == act.SourceNode.ObjectMeta.Name {
		color.Red("失败")
		return fmt.Errorf("source and distination are same node")
	}
	color.Green("正常")

	var pvcp = act.PvcPairs[index]

	// check pv
	var pvs = pvcp.pv.Status.Phase
	fmt.Printf("    检查PV状态是否为有效绑定 ")
	if pvs != v1.VolumeBound {
		color.Red("失败")
		return fmt.Errorf("we except pv %s status be Bound, but we got %s", pvcp.pv.ObjectMeta.Name, pvs)
	}
	color.Green("正常")

	// check pvc
	var pvcs = pvcp.pvc.Status.Phase
	fmt.Printf("    检查PVC状态是否为有效绑定 ")
	if pvcs != v1.ClaimBound {
		color.Red("失败")
		return fmt.Errorf("we expect pvc %s status be Bound, but we got %s", pvcp.pvc.ObjectMeta.Name, pvs)
	}
	color.Green("正常")

	// check pod
	var ps = act.Pod.Status.Phase
	fmt.Printf("    检查Pod状态是否正常 ")
	if ps != v1.PodRunning && ps != v1.PodSucceeded {
		color.Red("失败")
		return fmt.Errorf("we expect node %s status be Running or Succeeded, bu we got %s", act.Pod.ObjectMeta.Name, ps)
	}
	color.Green("正常")

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
