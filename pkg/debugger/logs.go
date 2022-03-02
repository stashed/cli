package debugger

import (
	"gomodules.xyz/go-sh"
	core "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func showLogs(pod *core.Pod, args ...string) error {
	klog.Infof("\n\n\n\n\n\n===============[ Logs from pod: %s ]===============", pod.Name)
	cmdArgs := []string{"logs", "-n", pod.Namespace, pod.Name}
	cmdArgs = append(cmdArgs, args...)
	if err := sh.Command("kubectl", cmdArgs).Run(); err != nil {
		return err
	}
	return nil
}
