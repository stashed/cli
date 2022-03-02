package debugger

import "stash.appscode.dev/apimachinery/apis/stash/v1beta1"

func (opt *DebugOptions) debugSidecar(targetRef v1beta1.TargetRef, container string) error {
	workloadPods, err := opt.getWorkloadPods(targetRef)
	if err != nil {
		return err
	}
	for _, workloadPods := range workloadPods.Items {
		if err := showLogs(&workloadPods, "-c", container); err != nil {
			return err
		}
	}
	return nil
}
