package debugger

import "stash.appscode.dev/apimachinery/apis/stash/v1beta1"

func (opt *DebugOptions) DebugBackupConfig(backupConfig *v1beta1.BackupConfiguration) error {
	if backupConfig.Status.Phase == v1beta1.BackupInvokerReady {
		backupSessions, err := opt.getOwnedBackupSessions(backupConfig)
		if err != nil {
			return err
		}
		for _, backupSession := range backupSessions {
			if err := opt.debugBackupSession(&backupSession, []v1beta1.BackupConfigurationTemplateSpec{backupConfig.Spec.BackupConfigurationTemplateSpec}); err != nil {
				return err
			}
		}
	} else {
		if err := describeObject(backupConfig, v1beta1.ResourceKindBackupConfiguration); err != nil {
			return err
		}
	}
	return nil
}
