package pv

import (
	"fmt"
	"strings"

	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"go.zoe.im/kops/pkg/utils"
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
}

// NewManager create a new manager
func NewManager() *Manager {
	return &Manager{
		Config: NewConfig(),
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

	// if keys is empty, we are trying to list all pv(local) information
	if len(keys) == 0 && m.Config.Source == "" && m.Config.Target == "" {
		fmt.Println("[INFO] 列出所有PV")
		// list all pv
		pvs, err := listPV(IsLocalPV())
		if err != nil {
			fmt.Println("[ERROR] 列出PV错误:", err)
			return
		}

		for _, i := range pvs {
			// display more informations, just print
			fmt.Println(i.Name)
		}
		return
	}

	// if source is empty and source is not empty
	if m.Config.Target == "" && m.Config.Source != "" && len(keys) > 0 {

		var source = m.Config.Source
		var _parts = strings.Split(m.Config.Source, "@")
		if len(_parts) == 2 {
			source = _parts[1]
		}

		// list all pods with filter
		pods, err := listPods(PodNameLike(keys...), WithPV(), PodOnHost(source))
		if err != nil {
			fmt.Println("[ERROR] 列出Pod错误:", err)
			return
		}

		fmt.Println("[INFO] 移动节点", source, "上", len(pods), "个符合名称", keys, "的Pod，至当前节点")

	pod_loop:
		for _, pod := range pods {
			// display more informations, just print
			fmt.Println("\n[INFO] 将Pod", pod.Name, "移动至当前节点")

			// create action job to exectud

			// find the pvc of pod
			var pvcps = []*PVCPair{}
			for _, v := range pod.Spec.Volumes {
				if v.PersistentVolumeClaim == nil {
					continue
				}

				// get pvc
				// check if is the local pv
				pvc, err := getPVC(v.PersistentVolumeClaim.ClaimName)
				if err != nil {
					fmt.Println("[ERROR] 操作终止，因为在获取PVC", v.PersistentVolumeClaim.ClaimName, "时出现错误:", err)
					continue pod_loop
				}

				// get pv
				pv, err := getPV(pvc.Spec.VolumeName)
				if err != nil {
					fmt.Println("[ERROR] 操作终止，因为在获取PV", pvc.Spec.VolumeName, "时出现错误:", err)
					continue pod_loop
				}

				pvcps = append(pvcps, &PVCPair{
					pv:  *pv,
					pvc: *pvc,
				})
			}

			// get current node
			current, err := getNode(m.hostname)
			if err != nil {
				fmt.Println("[ERROR] 获取当前节点信息出错:", err)
				return
			}

			// get source node: split password and username
			// split with :
			sources, err := listNode(IsHost(source))
			if err != nil {
				fmt.Println("[ERROR] 获取目标节点", source, "信息出错:", err)
				return
			}

			if len(sources) != 1 {
				fmt.Println("[ERROR] 目标节点", source, "需要有唯一1个,", "但是发现", len(sources), "个")
				return
			}

			// TODO: auto replace hostname with ip
			for _, adr := range sources[0].Status.Addresses {
				if adr.Type == "InternalIP" {
					source = adr.Address
				}
			}

			action := ActionConfig{
				Pod:        pod,
				PvcPairs:   pvcps,
				SourceNode: sources[0],
				TargetNode: *current,
				srcHost:    source,
				m:          m,
			}

			if !m.Config.Yes {
				r := utils.Ask("🚀   即将执行的操作很危险，是否继续?(N/y)")
				if r != "y\n" {
					return
				}
			}

			err = action.Run()
			if err == ErrCancel {
				fmt.Println("[WARN]", pod.Name, "迁移被取消")
				continue
			}
			if err != nil {
				fmt.Println("[ERROR]", pod.Name, "迁移失败:", err)
				return
			}
			fmt.Println("[INFO]", pod.Name, "迁移成功")
		}

		fmt.Println("\n[INFO] 全部任务执行结束")
		return
	}

	// if target is empty, mv pods to current node
	// init the client set
	// podlist, err := podclient.List(metav1.ListOptions{})
	// if err != nil {
	// 	fmt.Println("list pods error:", err)
	// }

	// for _, i := range podlist.Items {
	// 	fmt.Println(i.Name)
	// }

	fmt.Println("[ERROR] 参数不明确")
}
