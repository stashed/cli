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
	"context"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	core_util "kmodules.xyz/client-go/core/v1"
)

func (opt *options) debugBackupSession(backupSession *v1beta1.BackupSession, members []v1beta1.BackupConfigurationTemplateSpec) error {
	if err := opt.describeObject(backupSession.Name, v1beta1.ResourceKindBackupSession); err != nil {
		return err
	}
	if backupSession.Status.Phase == v1beta1.BackupSessionPending || backupSession.Status.Phase == v1beta1.BackupSessionSkipped {
		return nil
	}
	for _, member := range members {
		klog.Infof("\n\n\n\n\n\n==================[ Debugging backup for target %s %s/%s ]==================", member.Target.Ref.Kind, opt.namespace, member.Target.Ref.Name)
		if util.BackupModel(member.Target.Ref.Kind, member.Task.Name) == apis.ModelSidecar {
			if err := opt.debugSidecar(member.Target.Ref, apis.StashContainer); err != nil {
				return err
			}
		} else {
			if err := opt.debugJobs(backupSession); err != nil {
				return err
			}
		}
	}
	return nil
}

func (opt *options) getOwnedBackupSessions(backupInvoker metav1.Object) ([]v1beta1.BackupSession, error) {
	bsList, err := opt.stashClient.StashV1beta1().BackupSessions(backupInvoker.GetNamespace()).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var ownedBackupsessions []v1beta1.BackupSession
	for i := range bsList.Items {
		if owned, _ := core_util.IsOwnedBy(&bsList.Items[i], backupInvoker); owned {
			ownedBackupsessions = append(ownedBackupsessions, bsList.Items[i])
		}
	}
	return ownedBackupsessions, nil
}
