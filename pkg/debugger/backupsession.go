package debugger

import (
	"context"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	core_util "kmodules.xyz/client-go/core/v1"
)

func (opt *DebugOptions) debugBackupSession(backupSession *v1beta1.BackupSession, members []v1beta1.BackupConfigurationTemplateSpec) error {
	if backupSession.Status.Phase == v1beta1.BackupSessionFailed {
		if err := describeObject(backupSession, v1beta1.ResourceKindBackupSession); err != nil {
			return err
		}
		for _, member := range members {
			klog.Infof("\n\n\n\n\n\n===============[ Debugging backup for %s ]===============", member.Target.Ref.Name)
			backupModel := util.BackupModel(member.Target.Ref.Kind)
			if backupModel == apis.ModelSidecar {
				if err := opt.debugSidecar(member.Target.Ref, apis.StashContainer); err != nil {
					return err
				}
			} else if backupModel == apis.ModelCronJob {
				if err := opt.debugJob(backupSession); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (opt *DebugOptions) getOwnedBackupSessions(backupInvoker metav1.Object) ([]v1beta1.BackupSession, error) {
	backupSessionList, err := opt.StashClient.StashV1beta1().BackupSessions(backupInvoker.GetNamespace()).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var ownedBackupsessions []v1beta1.BackupSession
	for _, backupSession := range backupSessionList.Items {
		owned, _ := core_util.IsOwnedBy(&backupSession, backupInvoker)
		if owned {
			ownedBackupsessions = append(ownedBackupsessions, backupSession)
		}
	}
	return ownedBackupsessions, nil
}
