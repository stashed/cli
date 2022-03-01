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

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func NewCmdCopyBackupConfiguration() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "backupconfig",
		Short:             `Copy BackupConfiguration from one namespace to another namespace`,
		Long:              `Copy BackupConfiguration with respective Repository and Secret if they are not present in the target namespace`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("BackupConfiguration name is not provided")
			}

			backupConfigName := args[0]
			// get source BackupConfiguration and respective Repository and Secret from current namespace
			// if found then copy the BackupConfiguration, Repository and Secret to the destination namespace
			return ensureBackupConfiguration(backupConfigName)
		},
	}

	return cmd
}

func ensureBackupConfiguration(name string) error {
	// get resource BackupConfiguration
	backupConfig, err := stashClient.StashV1beta1().BackupConfigurations(srcNamespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	// Repository holds the backend information, In Restic driver mechanism, Repository is used to backup.
	// For that need to insure Repository and Secret
	if backupConfig.Spec.Driver != v1beta1.VolumeSnapshotter {
		// ensure Repository and Secret
		err = ensureRepository(backupConfig.Spec.Repository.Name)
		if err != nil {
			return err
		}
	}
	// copy the BackupConfiguration to the destination namespace
	meta := metav1.ObjectMeta{
		Name:        backupConfig.Name,
		Namespace:   dstNamespace,
		Labels:      backupConfig.Labels,
		Annotations: backupConfig.Annotations,
	}
	_, err = createBackupConfiguration(backupConfig, meta)
	if err != nil {
		return err
	}

	klog.Infof("BackupConfiguration %s/%s has been copied to %s namespace successfully.", srcNamespace, backupConfig.Name, dstNamespace)
	return err
}
