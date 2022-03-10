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

	"stash.appscode.dev/cli/pkg/debugger"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/util/templates"
)

var debugBackupExample = templates.Examples(`
		# Debug a BackupConfigration
		stash debug backup --namespace=<namespace> --backupconfig=<backupconfiguration-name>
        stash debug backup --namespace=demo --backupconfig=sample-mongodb-backup`)

func NewCmdDebugBackup() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "backup",
		Short:             `Debug backup`,
		Long:              `Debug common Stash backup issues`,
		Example:           debugBackupExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dbgr := debugger.NewDebugger(kubeClient, stashClient, aggrClient, namespace)
			if backupConfig == "" && backupBatch == "" {
				return fmt.Errorf("neither BackupConfiguration nor BackupBatch name has been provided")
			}
			if err := dbgr.ShowVersionInformation(); err != nil {
				return err
			}

			if backupConfig != "" {
				bc, err := stashClient.StashV1beta1().BackupConfigurations(namespace).Get(context.TODO(), backupConfig, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if err := dbgr.DebugBackupConfig(bc); err != nil {
					return err
				}
			} else {
				bb, err := stashClient.StashV1beta1().BackupBatches(namespace).Get(context.TODO(), backupBatch, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if err := dbgr.DebugBackupBatch(bb); err != nil {
					return err
				}

			}
			return nil
		},
	}
	cmd.Flags().StringVar(&backupConfig, "backupconfig", backupConfig, "Name of the BackupConfiguration to debug")
	cmd.Flags().StringVar(&backupBatch, "backupbatch", backupBatch, "Name of the BackupBatch to debug")
	return cmd
}
