package pv

import (
	"errors"
	"fmt"
	"time"

	"github.com/fatih/color"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func HandlerDeletePvc(act *ActionConfig) error {
	// modify recycle of pv?
	// **make sure data has been synced!important. rdiff?
	fmt.Printf("[%d] 删除PVC\n", act.CurrentStep)

	for idx := range act.Items {
		var item = act.Items[idx]
		fmt.Printf("    调整第 %d 个 PV %s 的回收策略为 Retain ", idx, item.OldPv.Name)
		_, err := pvclient.Patch(item.OldPv.Name, types.MergePatchType, PatchPVRecycle)
		if err != nil {
			color.Red("失败")
			fmt.Printf("    error: %s\n", err)
			return err
		}
		// wait for recycle pf pv has been changed to retain
		// FIXME: use watch
		for {
			time.Sleep(time.Second)
			pv, err := pvclient.Get(item.OldPv.Name, metav1.GetOptions{})
			if err == nil {
				if pv.Spec.PersistentVolumeReclaimPolicy == "Retain" {
					color.Green("成功")
					break
				}
			}
		}

		fmt.Printf("    删除PVC %s ", item.OldPvc.Name)
		err = pvcclient.Delete(item.OldPvc.Name, nil)
		if err != nil {
			color.Red("失败")
			return err
		}
		color.Green("成功")
	}

	return nil
}

func HandlerDeletePod(act *ActionConfig) error {
	// **make sure pvc status to terminating!important
	fmt.Printf("[%d] 删除Pod %s ", act.CurrentStep, act.Pod.Name)
	err := podclient.Delete(act.Pod.Name, nil)
	if err != nil {
		color.Red("失败")
		return err
	}
	color.Green("成功")

	fmt.Printf("    等待PVC被删除 ")
	// FIXME: use watch
	var c = 0
	for {
		fmt.Printf(".")
		time.Sleep(time.Second)
		var _deleted = true
		for idx := range act.Items {
			var item = act.Items[idx]
			_, err := pvcclient.Get(item.OldPvc.Name, metav1.GetOptions{})
			if err != nil {
				item.OldPvcDeleted = true
			} else {
				_deleted = false
				break
			}
		}

		if _deleted {
			color.Green("完成")
			act.PodDeleted = true
			return nil
		}

		c += 1
		if act.m.Config.Wait > 0 && c == act.m.Config.Wait {
			color.Yellow("\n    超时，可能删除失败")
			return errors.New("删除失败")
		}
	}
}

func HandlerDeletePv(act *ActionConfig) error {
	// TODO:
	fmt.Printf("[%d] 删除原PV ", act.CurrentStep)
	color.Yellow("跳过")
	return nil
}

func HandlerRestartPod(act *ActionConfig) error {
	// timeout.
	// TODO: we need to roll back all actions?

	// actually pvc status is BOUND
	// pvcclient.
	fmt.Printf("[%d] 重启Pod %s ", act.CurrentStep, act.Pod.Name)

	podclient.Delete(act.Pod.Name, nil)
	color.Green("成功")
	return nil
}