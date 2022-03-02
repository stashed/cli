package debugger

import (
	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
)

func (opt *DebugOptions) DebugRestoreSession(restoreSession *v1beta1.RestoreSession) error {
	if restoreSession.Status.Phase != v1beta1.RestoreSucceeded {
		if err := describeObject(restoreSession, v1beta1.ResourceKindRestoreSession); err != nil {
			return err
		}
		restoreModel := util.RestoreModel(restoreSession.Spec.Target.Ref.Kind)
		if restoreModel == apis.ModelSidecar {
			if err := opt.debugSidecar(restoreSession.Spec.Target.Ref, apis.StashInitContainer); err != nil {
				return err
			}
		} else if restoreModel == apis.ModelCronJob {
			if err := opt.debugJob(restoreSession); err != nil {
				return err
			}
		}
	}
	return nil
}
