package pv

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"go.zoe.im/kops/pkg/utils"
	"github.com/fatih/color"
)

var (
	clientset *kubernetes.Clientset

	coreclient corev1.CoreV1Interface
	nodeclient corev1.NodeInterface
	podclient  corev1.PodInterface // special namespace
	pvclient   corev1.PersistentVolumeInterface
	pvcclient  corev1.PersistentVolumeClaimInterface // special namespace
)

// Manager controll things
type Manager struct {
	Config *Config

	hostname string

	fns []ActionHandler
}

// NewManager create a new manager
func NewManager() *Manager {
	return &Manager{
		Config: NewConfig(),
		fns: []ActionHandler{
			HandlerCheck,
			HandlerSync,
			HandlerDeletePvc,
			HandlerDeletePod,
			HandlerSync,
			HandlerDeletePv,
			HandlerCreatePv,
			HandlerCreatePvc,
			HandlerRestartPod,
		},
	}
}

func (m *Manager) init() {

	// default namespace, TODO:
	namespace := m.Config.Namespace

	clientset = utils.GetKubeClient()
	coreclient = clientset.CoreV1()

	// create all clients
	nodeclient = coreclient.Nodes()
	podclient = coreclient.Pods(namespace)
	pvclient = coreclient.PersistentVolumes()
	pvcclient = coreclient.PersistentVolumeClaims(namespace)

	var err error
	m.hostname, err = utils.Hostname()
	if err != nil {
		panic("获取当前Hostname错误")
	}
}

func (m *Manager) dumps(act *ActionConfig) {
	var log = fmt.Sprintf("kops-%s-%d", act.Pod.Name, time.Now().Unix())
	f, err := os.Create(log)
	if err != nil {
		color.Red("[DUMPS] 打开文件错误: %s", err)
		return
	}
	var enc = json.NewEncoder(f)
	err = enc.Encode(act)
	if err != nil {
		color.Red("[DUMPS] 写入文件错误: %s", err)
	}
	color.Yellow("[DUMPS] 保留线程在文件: %s", log)
}

func (m *Manager) recover() error {
	f, err := os.Open(m.Config.LogFile)
	fmt.Sprintf("恢复运行 %s \n", m.Config.LogFile)
	if err != nil {
		color.Red("[RECOVER] 打开文件错误: %s", err)
		return err
	}
	var dec = json.NewDecoder(f)
	var act  = &ActionConfig{}
	err = dec.Decode(act)
	if err != nil {
		color.Red("[RECOVER] 反序列化数据失败: %s", err)
		return err
	}
	if m.Config.Step > 0 {
		act.CurrentStep = m.Config.Step
		color.Yellow("手动设置至第 %d 步", act.CurrentStep)
	}
	act.m = m
	// strart to run
	return act.Run()
}

// FIXME: make this more beautify
func (m *Manager) precheck() error {
	return nil
}

// Start process
// 0. nothing PANIC or list all pv
// 1. pods name, mv [pods] to current node [x]
// 2. distination, mv pods from current node to distination [x]
// 3. both pods name and distination, mv [pods] to distination
// 4. backup data [x]
//
// First of all, we need to implement transport [pods] to current node
func (m *Manager) Start(keys ...string) {

	// init
	m.init()

	// recovery from file
	if m.Config.LogFile != "" {
		err := m.recover()
		fmt.Sprintf("从该日志 %s 中恢复运行", m.Config.LogFile)
		if err != nil {
			color.Red("运行失败: %s", err)
			return
		}
		color.Green("运行成功")
		return
	}

	if len(keys) == 0 {
		color.Red("需提供至少一个Pod以用于迁移")
		return
	}

	// create filters
	var filters = []PodFilter{
		PodNameLike(keys...), WithPV(),
	}

	if m.Config.Target != "" {
		color.Yellow("暂不支持指定迁移目标机器，仅支持从远程机器迁移至本地")
		return
	}

	var source = m.Config.Source
	if m.Config.Source != "" {
		var _parts = strings.Split(m.Config.Source, "@")
		if len(_parts) == 2 {
			source = _parts[1]
		}
		filters = append(filters, PodOnHost(source))
	}

	// list all pods with filter
	pods, err := listPods(filters...)
	if err != nil {
		color.Red("列出Pod错误: %s", err)
		return
	}

	fmt.Printf("移动节点 %s 上 %d 个符合名称 %s 的Pod至当前节点 ", source, len(pods), keys)
	if m.Config.OnlySync {
		color.Yellow("[仅同步数据模式]")
	} else {
		fmt.Println()
	}

pod_loop:
	for _, pod := range pods {
		// display more informations, just print
		fmt.Println("\n将Pod", pod.Name, "移动至当前节点")

		// create action job to exectud

		// find the pvc of pod
		var pvcps = []*PVItem{}
		for _, v := range pod.Spec.Volumes {
			if v.PersistentVolumeClaim == nil {
				continue
			}

			// get pvc
			// check if is the local pv
			pvc, err := getPVC(v.PersistentVolumeClaim.ClaimName)
			if err != nil {
				color.Red("操作终止，因为在获取PVC %s 时出现错误: %s", v.PersistentVolumeClaim.ClaimName, err)
				continue pod_loop
			}

			// get pv
			pv, err := getPV(pvc.Spec.VolumeName)
			if err != nil {
				color.Red("操作终止，因为在获取PV %s 时出现错误", pvc.Spec.VolumeName, err)
				continue pod_loop
			}

			pvcps = append(pvcps, &PVItem{
				OldPv:  pv,
				OldPvc: pvc,
			})
		}

		// pod.Spec.NodeName

		// get current node
		current, err := getNode(m.hostname)
		if err != nil {
			color.Red("获取当前节点信息出错: %s", err)
			return
		}

		// get source node: split password and username
		// split with :
		sources, err := listNode(IsHost(pod.Spec.NodeName))
		if err != nil {
			color.Red("获取目标节点 %s 信息出错: %s", pod.Spec.NodeName, err)
			return
		}

		if len(sources) != 1 {
			color.Red("源节点 %s需要有唯一1个, 但是发现 %s 个", pod.Spec.NodeName, len(sources))
			return
		}

		// TODO: auto replace hostname with ip
		for _, adr := range sources[0].Status.Addresses {
			if adr.Type == "InternalIP" {
				source = adr.Address
			}
		}

		action := &ActionConfig{
			Pod:        pod,
			Items:      pvcps,
			SourceNode: sources[0],
			TargetNode: *current,
			SrcHost:    source,
			m:          m,
		}

		if !m.Config.Yes && !m.Config.OnlySync {
			r := utils.Ask("🚀 即将执行的操作很危险，是否继续?(N/y)")
			if r != "y\n" {
				return
			}
		}

		err = action.Run()
		fmt.Printf("Pod %s 迁移 ", pod.Name)
		if err == ErrCancel {
			color.Yellow("取消")
			continue
		}
		if err != nil {
			color.Red("失败")
			color.Red("[ERROR] %s", err)
			// dumps data
			m.dumps(action)
			return
		}
		color.Green("成功")
	}

	color.Green("\n全部任务执行结束")
	return
}