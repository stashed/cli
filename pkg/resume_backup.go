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

package pkg

import (
	"context"
	"fmt"

	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	resumeBackupExample = templates.Examples(`
		# Pause a BackupConfigration
		stash resume backup --namespace=<namespace> --backup-config=<backup-configuration-name>
        stash resume backup --backup-config=asample-mongodb -n demo`)
)

func NewCmdResumeBackup() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "backup",
		Short:             `resume backup`,
		Long:              `resume backup by patching Backupconfiguration/BackupBatch`,
		Example:           resumeBackupExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if backupConfig == "" && backupBatch == "" {
				return fmt.Errorf("BackupConfiguration/BackupBatch name has been not provided")
			}
			var err error
			if backupConfig != "" {
				err = resumeBackupconfiguration()
				if err == nil {
					klog.Infof("BackupConfiguration %s/%s has been resumed successfully.", namespace, backupConfig)
				}
			}
			if backupBatch != "" {
				err = resumeBackupBatch()
				if err == nil {
					klog.Infof("BackupBatch %s/%s has been resumed successfully.", namespace, backupBatch)
				}
			}
			return err
		},
	}

	return cmd
}

func resumeBackupconfiguration() error {
	bc, err := stashClient.StashV1beta1().BackupConfigurations(namespace).Get(context.TODO(), backupConfig, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, _, err = v1beta1_util.PatchBackupConfiguration(
		context.TODO(),
		stashClient.StashV1beta1(),
		bc,
		func(in *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
			in.Spec.Paused = false
			return in
		},
		metav1.PatchOptions{},
	)
	return err
}
func resumeBackupBatch() error {
	bb, err := stashClient.StashV1beta1().BackupBatches(namespace).Get(context.TODO(), backupBatch, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, _, err = v1beta1_util.PatchBackupBatch(
		context.TODO(),
		stashClient.StashV1beta1(),
		bb,
		func(in *v1beta1.BackupBatch) *v1beta1.BackupBatch {
			in.Spec.Paused = false
			return in
		},
		metav1.PatchOptions{},
	)
	return err
}
