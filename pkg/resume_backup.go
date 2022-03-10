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
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/util/templates"
)

var resumeBackupExample = templates.Examples(`
		# Resume a BackupConfigration
		stash resume backup --namespace=<namespace> --backupconfig=<backupconfiguration-name>
        stash resume backup --namespace=demo --backupconfig=sample-mongodb-backup`)

func NewCmdResumeBackup() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "backup",
		Short:             `Resume backup`,
		Long:              `Resume backup by setting "paused" field of BackupConfiguration/BackupBatch to "false"`,
		Example:           resumeBackupExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if backupConfig == "" && backupBatch == "" {
				return fmt.Errorf("neither BackupConfiguration nor BackupBatch name has been provided")
			}

			if backupConfig != "" {
				if err := setBackupConfigurationPausedField(false); err != nil {
					return err
				}
				klog.Infof("BackupConfiguration %s/%s has been resumed successfully.", namespace, backupConfig)
			} else {
				if err := setBackupBatchPausedField(false); err != nil {
					return err
				}
				klog.Infof("BackupBatch %s/%s has been resumed successfully.", namespace, backupBatch)
			}
			return nil
		},
	}

	return cmd
}
