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

var debugRestoreExample = templates.Examples(`
		# Debug a RestoreSession
		stash debug restore --namespace=<namespace> --restoresession=<restoresession-name>
       stash debug restore --namespace=demo --restoresession=sample-mongodb-restore`)

func NewCmdDebugRestore() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "restore",
		Short:             `Debug restore`,
		Long:              `Show debugging information for restore process`,
		Example:           debugRestoreExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dbgr := debugger.NewDebugger(kubeClient, stashClient, aggrClient, namespace)
			if restoreSession == "" && restoreBatch == "" {
				return fmt.Errorf("neither RestoreSession nor RestoreBatch name has been provided")
			}
			if err := dbgr.ShowVersionInformation(); err != nil {
				return err
			}
			if restoreSession != "" {
				rs, err := stashClient.StashV1beta1().RestoreSessions(namespace).Get(context.TODO(), restoreSession, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if err := dbgr.DebugRestoreSession(rs); err != nil {
					return err
				}
			} else {
				rb, err := stashClient.StashV1beta1().RestoreBatches(namespace).Get(context.TODO(), restoreBatch, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if err := dbgr.DebugRestoreBatch(rb); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&restoreSession, "restoresession", backupConfig, "Name of the RestoreSession to debug")
	cmd.Flags().StringVar(&restoreBatch, "restorebatch", backupBatch, "Name of the RestoreBatch to debug")
	return cmd
}
