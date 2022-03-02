package debugger

import (
	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"

	"k8s.io/klog/v2"
)

func (opt *DebugOptions) DebugRestoreBatch(restoreBatch *v1beta1.RestoreBatch) error {
	if restoreBatch.Status.Phase != v1beta1.RestoreSucceeded {
		if err := describeObject(restoreBatch, v1beta1.ResourceKindRestoreBatch); err != nil {
			return err
		}
		for _, member := range restoreBatch.Spec.Members {
			klog.Infof("\n\n\n\n\n\n===============[ Debugging restore for %s ]===============", member.Target.Ref.Name)
			restoreModel := util.BackupModel(member.Target.Ref.Kind)
			if restoreModel == apis.ModelSidecar {
				if err := opt.debugSidecar(member.Target.Ref, apis.StashInitContainer); err != nil {
					return err
				}
			} else if restoreModel == apis.ModelCronJob {
				if err := opt.debugJob(restoreBatch); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
