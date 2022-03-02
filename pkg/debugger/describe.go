package debugger

import (
	"gomodules.xyz/go-sh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func describeObject(object metav1.Object, resourceKind string) error {
	klog.Infof("\n\n\n\n\n\n===============[ Describing %s: %s ]===============", resourceKind, object.GetName())
	if err := sh.Command("kubectl", "describe", resourceKind, "-n", object.GetNamespace(), object.GetName()).Run(); err != nil {
		return err
	}
	return nil
}
