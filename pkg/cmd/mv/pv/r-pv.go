package pv

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PVFilter func(pv v1.PersistentVolume) bool

func IsLocalPV() PVFilter {
	return func(pv v1.PersistentVolume) bool {
		return pv.Spec.Local != nil
	}
}

func PVNameLike(patterns ...string) PVFilter {
	return func(pv v1.PersistentVolume) bool {
		// ORs
		for _, i := range patterns {
			if i == "*" || i == pv.Name {
				return true
			}
		}
		return false
	}
}

// get pv by name
func getPV(name string) (*v1.PersistentVolume, error) {
	return pvclient.Get(name, metav1.GetOptions{})
}

// listPV list all pvs
func listPV(filters ...func(v1.PersistentVolume) bool) ([]v1.PersistentVolume, error) {

	// list all pv
	r, err := pvclient.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items = []v1.PersistentVolume{}
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

var (
	tplPVList = `{{.Name}}`
)
