package pv

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodFilter func(pod v1.Pod) bool

func PodNameLike(patterns ...string) PodFilter {
	return func(pod v1.Pod) bool {
		for _, i := range patterns {
			if i == "*" || i == pod.Name {
				return true
			}
		}
		return false
	}
}

func PodOnHost(host string) PodFilter {
	return func(pod v1.Pod) bool {
		return pod.Spec.NodeName == host || pod.Status.HostIP == host
	}
}

// with local pv?
func WithPV() PodFilter {
	return func(pod v1.Pod) bool {
		for _, v := range pod.Spec.Volumes {
			if v.PersistentVolumeClaim != nil {
				return true
			}
		}
		return false
	}
}

// list all pods
func listPods(filters ...PodFilter) ([]v1.Pod, error) {
	r, err := podclient.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items = []v1.Pod{}

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