package debugger

import (
	"strings"

	"gomodules.xyz/go-sh"
	"k8s.io/klog/v2"
)

func (opt *DebugOptions) ShowVersionInformation() error {
	klog.Infoln("\n\n\n===============[ Kubernetes Version ]===============")
	if err := sh.Command("kubectl", "version", "--short").Run(); err != nil {
		return err
	}
	pod, err := opt.GetOperatorPod()
	if err != nil {
		return err
	}
	var stashBinary string
	if strings.Contains(pod.Name, "stash-enterprise") {
		stashBinary = "/stash-enterprise"
	} else {
		stashBinary = "/stash"
	}
	klog.Infoln("\n\n\n===============[ Stash Version ]===============")
	if err := sh.Command("kubectl", "exec", "-it", "-n", pod.Namespace, pod.Name, "-c", "operator", "--", stashBinary, "version").Run(); err != nil {
		return err
	}
	return nil
}
