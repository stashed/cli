package debugger

import (
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
)

func (opt *DebugOptions) DebugBackupBatch(backupBatch *v1beta1.BackupBatch) error {
	if backupBatch.Status.Phase == v1beta1.BackupInvokerReady {
		backupSessions, err := opt.getOwnedBackupSessions(backupBatch)
		if err != nil {
			return err
		}
		for _, backupSession := range backupSessions {
			if err := opt.debugBackupSession(&backupSession, backupBatch.Spec.Members); err != nil {
				return err
			}
		}
	} else {
		if err := describeObject(backupBatch, v1beta1.ResourceKindBackupBatch); err != nil {
			return err
		}
	}
	return nil
}
