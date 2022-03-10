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

var pauseBackupExample = templates.Examples(`
		# Pause a BackupConfigration
		stash pause backup --namespace=<namespace> --backupconfig=<backupconfiguration-name>
        stash pause backup --namespace=demo --backupconfig=sample-mongodb-backup`)

func NewCmdPauseBackup() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "backup",
		Short:             `Pause backup`,
		Long:              `Pause backup by setting "paused" field of BackupConfiguration/BackupBatch to "true"`,
		Example:           pauseBackupExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if backupConfig == "" && backupBatch == "" {
				return fmt.Errorf("neither BackupConfiguration nor BackupBatch name has been provided")
			}
			if backupConfig != "" {
				if err := setBackupConfigurationPausedField(true); err != nil {
					return err
				}
				klog.Infof("BackupConfiguration %s/%s has been paused successfully.", namespace, backupConfig)
			} else {
				if err := setBackupBatchPausedField(true); err != nil {
					return err
				}
				klog.Infof("BackupBatch %s/%s has been paused successfully.", namespace, backupBatch)
			}
			return nil
		},
	}
	return cmd
}

func setBackupConfigurationPausedField(value bool) error {
	bc, err := stashClient.StashV1beta1().BackupConfigurations(namespace).Get(context.TODO(), backupConfig, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, _, err = v1beta1_util.PatchBackupConfiguration(
		context.TODO(),
		stashClient.StashV1beta1(),
		bc,
		func(in *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
			in.Spec.Paused = value
			return in
		},
		metav1.PatchOptions{},
	)
	return err
}

func setBackupBatchPausedField(value bool) error {
	bb, err := stashClient.StashV1beta1().BackupBatches(namespace).Get(context.TODO(), backupBatch, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, _, err = v1beta1_util.PatchBackupBatch(
		context.TODO(),
		stashClient.StashV1beta1(),
		bb,
		func(in *v1beta1.BackupBatch) *v1beta1.BackupBatch {
			in.Spec.Paused = value
			return in
		},
		metav1.PatchOptions{},
	)
	return err
}
