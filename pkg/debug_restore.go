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

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
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
			if restoreSession == "" && restoreBatch == "" {
				return fmt.Errorf("neither RestoreSession nor RestoreBatch name has been provided")
			}

			if err := showVersionInformation(); err != nil {
				return err
			}

			if restoreSession != "" {
				rs, err := stashClient.StashV1beta1().RestoreSessions(namespace).Get(context.TODO(), restoreSession, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if err := debugRestoreSession(rs); err != nil {
					return err
				}
			} else {
				rb, err := stashClient.StashV1beta1().RestoreBatches(namespace).Get(context.TODO(), restoreBatch, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if err := debugRestoreBatch(rb); err != nil {
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

func debugRestoreSession(restoreSession *v1beta1.RestoreSession) error {
	if restoreSession.Status.Phase != v1beta1.RestoreSucceeded {
		if err := describeObject(restoreSession, v1beta1.ResourceKindRestoreSession); err != nil {
			return err
		}
		restoreModel := util.RestoreModel(restoreSession.Spec.Target.Ref.Kind)
		if restoreModel == apis.ModelSidecar {
			if err := debugSidecar(restoreSession.Spec.Target.Ref, apis.StashInitContainer); err != nil {
				return err
			}
		} else if restoreModel == apis.ModelCronJob {
			if err := debugJob(restoreSession); err != nil {
				return err
			}
		}
	}
	return nil
}

func debugRestoreBatch(restoreBatch *v1beta1.RestoreBatch) error {
	if restoreBatch.Status.Phase != v1beta1.RestoreSucceeded {
		if err := describeObject(restoreBatch, v1beta1.ResourceKindRestoreBatch); err != nil {
			return err
		}
		for _, member := range restoreBatch.Spec.Members {
			klog.Infof("\n\n\n\n\n\n===============[ Debugging restore for %s ]===============", member.Target.Ref.Name)
			restoreModel := util.BackupModel(member.Target.Ref.Kind)
			if restoreModel == apis.ModelSidecar {
				if err := debugSidecar(member.Target.Ref, apis.StashInitContainer); err != nil {
					return err
				}
			} else if restoreModel == apis.ModelCronJob {
				if err := debugJob(restoreBatch); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
