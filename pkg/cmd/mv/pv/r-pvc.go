package pv

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// get pvc by name
func getPVC(name string) (*v1.PersistentVolumeClaim, error) {
	return pvcclient.Get(name, metav1.GetOptions{})
}
