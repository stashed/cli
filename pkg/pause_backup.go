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
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/util/templates"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	createBackupConfigrationExample = templates.Examples(`
		# Create a new repository
		stash create repository --namespace=<namespace> <repository-name> [Flag]
        stash create repository gcs-repo --namespace=demo --secret=gcs-secret --bucket=appscode-qa --prefix=/source/data --provider=gcs`)
)

func NewCmdPauseBackup() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "backup",
		Short:             `Pause backup`,
		Long:              `Pause backup by patching Backupconfiguration/BackupBatch`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if backupConfig == "" && backupBatch == "" {
				return fmt.Errorf("BackupConfiguration/BackupBatch name has been not provided")
			}
			var err error
			if backupConfig != "" {
				err = pauseBackupconfiguration()
				if err == nil {
					klog.Infof("BackupConfiguration %s/%s has been paused successfully.", namespace, backupConfig)
				}
			}
			if backupBatch != "" {
				err = pauseBackupBatch()
				if err == nil {
					klog.Infof("BackupBatch %s/%s has been paused successfully.", namespace, backupBatch)
				}
			}
			return err
		},
	}

	return cmd
}

func pauseBackupconfiguration() error {
	bc, err := stashClient.StashV1beta1().BackupConfigurations(namespace).Get(context.TODO(), backupConfig, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, _, err = v1beta1_util.PatchBackupConfiguration(
		context.TODO(),
		stashClient.StashV1beta1(),
		bc,
		func(in *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
			in.Spec.Paused = true
			return in
		},
		metav1.PatchOptions{},
	)
	return err
}
func pauseBackupBatch() error {
	bb, err := stashClient.StashV1beta1().BackupBatches(namespace).Get(context.TODO(), backupBatch, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, _, err = v1beta1_util.PatchBackupBatch(
		context.TODO(),
		stashClient.StashV1beta1(),
		bb,
		func(in *v1beta1.BackupBatch) *v1beta1.BackupBatch {
			in.Spec.Paused = true
			return in
		},
		metav1.PatchOptions{},
	)
	return err
}
