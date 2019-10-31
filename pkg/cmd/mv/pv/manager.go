package pv

import (
	"fmt"

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
	podclient  = coreclient.Pods(namespace)
	pvclient   = coreclient.PersistentVolumes()
	pvcclient  = coreclient.PersistentVolumeClaims(namespace)

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

	// if source is empty and target is not empty
	if m.Config.Target != "" && m.Config.Source == "" && len(keys) > 0 {

		fmt.Println("[INFO] 移动当前节点上符合名称", keys, "的Pod，至本节点", m.Config.Target)
		// list all pods
		pods, err := listPods(PodNameLike(keys...), PodOnHost(m.hostname))
		if err != nil {
			fmt.Println("[ERROR] 列出Pod错误:", err)
			return
		}
		
		for _, pod := range pods {
			// display more informations, just print
			fmt.Println("[INFO] 将Pod", pod.Name, "移动至目标节点")

			// create action job to exectud

			// get pvc

			// get pv

			// get node informations
		}

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
