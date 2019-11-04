package pv

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeFilter func(v1.Node) bool

func IsHost(name string) NodeFilter {
	return func(node v1.Node) bool {
		for _, dr := range node.Status.Addresses {
			if dr.Address == name {
				return true
			}
		}
		return false
	}
}

// get node by name
func getNode(name string) (*v1.Node, error) {
	return nodeclient.Get(name, metav1.GetOptions{})
}

// list all nodes
func listNode(filters ...NodeFilter) ([]v1.Node, error) {
	r, err := nodeclient.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items = []v1.Node{}

mainLoop:
	for _, i := range r.Items {
		for _, f := range filters {
			if !f(i) {
				continue mainLoop
			}
		}
		items = append(items, i)
	}
	
	return items, nil
}