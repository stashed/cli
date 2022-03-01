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
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	core_util "kmodules.xyz/client-go/core/v1"
)

func NewCmdTriggerBackup(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "trigger",
		Short:             `Trigger a backup`,
		Long:              `Trigger a backup by creating BackupSession`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("BackupConfiguration name not found")
			}
			backupConfigName := args[0]

			cfg, err := clientGetter.ToRESTConfig()
			if err != nil {
				return errors.Wrap(err, "failed to read kubeconfig")
			}
			namespace, _, err := clientGetter.ToRawKubeConfigLoader().Namespace()
			if err != nil {
				return err
			}

			client, err := cs.NewForConfig(cfg)
			if err != nil {
				return err
			}

			// get backupConfiguration
			backupConfig, err := client.StashV1beta1().BackupConfigurations(namespace).Get(context.TODO(), backupConfigName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			_, err = triggerBackup(backupConfig, client)
			return err
		},
	}

	return cmd
}

func triggerBackup(backupConfig *v1beta1.BackupConfiguration, client cs.Interface) (*v1beta1.BackupSession, error) {
	// create backupSession for backupConfig
	backupSession := &v1beta1.BackupSession{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: backupConfig.Name + "-",
			Namespace:    backupConfig.Namespace,
			Labels: map[string]string{
				apis.LabelApp:         apis.AppLabelStash,
				apis.LabelInvokerType: "BackupConfiguration",
				apis.LabelInvokerName: backupConfig.Name,
			},
		},
		Spec: v1beta1.BackupSessionSpec{
			Invoker: v1beta1.BackupInvokerRef{
				APIGroup: v1beta1.SchemeGroupVersion.Group,
				Kind:     v1beta1.ResourceKindBackupConfiguration,
				Name:     backupConfig.Name,
			},
		},
	}

	// set backupConfig as backupSession's owner
	owner := metav1.NewControllerRef(backupConfig, v1beta1.SchemeGroupVersion.WithKind(v1beta1.ResourceKindBackupConfiguration))
	core_util.EnsureOwnerReference(&backupSession.ObjectMeta, owner)

	// don't use createOrPatch here
	backupSession, err := client.StashV1beta1().BackupSessions(backupSession.Namespace).Create(context.TODO(), backupSession, metav1.CreateOptions{})
	if err != nil {
		return backupSession, err
	}
	klog.Infof("BackupSession %s/%s has been created successfully", backupSession.Namespace, backupSession.Name)
	return backupSession, nil
}
