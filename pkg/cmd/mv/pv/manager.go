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
}

// NewManager create a new manager
func NewManager() *Manager {
	return &Manager{
		Config: NewConfig(),
	}
}

func (m *Manager) init() {

 	// default namespace, TODO:
	namespace := ""

	clientset = utils.GetKubeClient()
	coreclient = clientset.CoreV1()

	// create all clients
	nodeclient = coreclient.Nodes()
	podclient  = coreclient.Pods(namespace)
	pvclient   = coreclient.PersistentVolumes()
	pvcclient  = coreclient.PersistentVolumeClaims(namespace)
}

// Start process
// 0. nothing PANIC or list all pv
// 1. pods name, mv [pods] to current node
// 2. distination, mv pods from current node to distination
// 3. both pods name and distination, mv [pods] to distination
//
// First of all, we need to implement transport [pods] to current node
func (m *Manager) Start(keys ...string) {

	// init 
	m.init()

	// if keys is empty, we are trying to list all pv(local) information
	if len(keys) == 0 && m.Config.Source == "" && m.Config.Target == "" {
		// list all pv
		pvs, err := listPV(IsLocalPV())
		if err != nil {
			fmt.Println("list all pv error:", err)
			return
		}
	
		for _, i := range pvs {
			// display more informations, just print
			fmt.Println(i.Name)
		}
		return
	}


	// init the client set
	// podlist, err := podclient.List(metav1.ListOptions{})
	// if err != nil {
	// 	fmt.Println("list pods error:", err)
	// }

	// for _, i := range podlist.Items {
	// 	fmt.Println(i.Name)
	// }
}
