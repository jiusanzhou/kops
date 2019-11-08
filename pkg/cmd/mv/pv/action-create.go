package pv

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HandlerCreatePvc(act *ActionConfig) error {
	fmt.Printf("[%d] 恢复创建PVC\n", act.CurrentStep)
	for idx := range act.Items {
		var item = act.Items[idx]
		var opvc = item.OldPvc

		fmt.Printf("    恢复第 %d 个创建PVC %s ", idx, opvc.Name)

		item.NewPvc = &v1.PersistentVolumeClaim{
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
				VolumeName:       item.NewPv.Name, // important
				StorageClassName: opvc.Spec.StorageClassName,
				VolumeMode:       opvc.Spec.VolumeMode,
				DataSource:       opvc.Spec.DataSource,
			},
		}
		_, err := pvcclient.Create(item.NewPvc)
		if err != nil {
			color.Red("失败")
			return err
		}
		color.Green("成功")
	}
	return nil
}

func HandlerCreatePv(act *ActionConfig) error {
	// 1. rename from the old one
	// 2. change the node affinity to the new node
	// 3. change the data path if need
	fmt.Printf("[%d] 在新节点上创建PV\n", act.CurrentStep)

	for idx := range act.Items {
		var item = act.Items[idx]
		var opv = item.OldPv
		item.NewPv = &v1.PersistentVolume{
			TypeMeta: opv.TypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:          withSuffixPVName(opv.ObjectMeta.Name, act.m.Config.Prefix),
				Namespace:     opv.ObjectMeta.Namespace,
				Labels:        opv.ObjectMeta.Labels,
				Annotations:   opv.ObjectMeta.Annotations,
				ClusterName:   opv.ObjectMeta.ClusterName,
				ManagedFields: opv.ObjectMeta.ManagedFields,
			},
			Spec: v1.PersistentVolumeSpec{
				AccessModes:                   opv.Spec.AccessModes,
				Capacity:                      opv.Spec.Capacity,
				PersistentVolumeSource:        v1.PersistentVolumeSource{
					Local: &v1.LocalVolumeSource{
						Path: item.TargetPath,
					},
				},
				PersistentVolumeReclaimPolicy: "Retain",
				StorageClassName:              opv.Spec.StorageClassName,
				MountOptions:                  opv.Spec.MountOptions,
				VolumeMode:                    opv.Spec.VolumeMode,
				NodeAffinity:                  opv.Spec.NodeAffinity,
			},
		}
		// TODO: just panic if empty???
		for i, term := range item.NewPv.Spec.NodeAffinity.Required.NodeSelectorTerms {
			for x, me := range term.MatchExpressions {
				if me.Operator == v1.NodeSelectorOpIn {
					me.Values = []string{act.TargetNode.ObjectMeta.Name} // new node
				}
				item.NewPv.Spec.NodeAffinity.Required.NodeSelectorTerms[i].MatchExpressions[x] = me
			}
			for x, mf := range term.MatchFields {
				if mf.Operator == v1.NodeSelectorOpIn {
					mf.Values = []string{act.TargetNode.ObjectMeta.Name} // new node
				}
				item.NewPv.Spec.NodeAffinity.Required.NodeSelectorTerms[i].MatchExpressions[x] = mf
			}
		}

		fmt.Printf("    创建第 %d 个创建PVC %s ", idx, item.NewPv.Name)

		// call api to create
		_, err := pvclient.Create(item.NewPv)
		if err != nil {
			color.Red("失败")
			return err
		}
		color.Green("成功")
	}

	return nil
}


func withSuffixPVName(name, suffix string) string {
	var splits = strings.Split(name, "-")
	var l = len(splits)
	if l >= 3 && splits[l-2] == suffix {
		// maybe this pv is a synced one, just replace the timestamp
		splits[l-1] = fmt.Sprintf("%d", time.Now().Unix())
		return strings.Join(splits, "-")
	} else {
		return fmt.Sprintf("%s-%s-%d", name, suffix, time.Now().Unix())
	}
}