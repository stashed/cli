/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package debugger

import (
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
)

func (opt *options) DebugBackupConfig(backupConfig *v1beta1.BackupConfiguration) error {
	if err := opt.describeObject(backupConfig.Name, v1beta1.ResourceKindBackupConfiguration); err != nil {
		return err
	}
	if err := opt.describeObject(backupConfig.Spec.Repository.Name, v1alpha1.ResourceKindRepository); err != nil {
		return err
	}

	if backupConfig.Status.Phase == v1beta1.BackupInvokerReady {
		backupSessions, err := opt.getOwnedBackupSessions(backupConfig)
		if err != nil {
			return err
		}
		for _, backupSession := range backupSessions {
			if backupSession.Status.Phase != v1beta1.BackupSessionSucceeded {
				if err := opt.debugBackupSession(&backupSession, []v1beta1.BackupConfigurationTemplateSpec{backupConfig.Spec.BackupConfigurationTemplateSpec}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
