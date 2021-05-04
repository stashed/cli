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

package docker

import (
	"path/filepath"

	"stash.appscode.dev/apimachinery/pkg/restic"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

func NewDeleteSnapshotCmd() *cobra.Command {
	var snapshotID string
	var cmd = &cobra.Command{
		Use:               "delete-snapshot",
		Short:             `Delete a snapshot from repository backend`,
		Long:              `Delete a snapshot from repository backend`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			setupOpt, err := ReadSetupOptionFromFile(filepath.Join(ConfigDir, SetupOptionsFile))
			if err != nil {
				return err
			}
			resticWrapper, err := restic.NewResticWrapper(*setupOpt)
			if err != nil {
				return err
			}
			// delete snapshots
			if _, err = resticWrapper.DeleteSnapshots([]string{snapshotID}); err != nil {
				return err
			}
			klog.Infof("Delete completed")
			return nil
		},
	}
	cmd.Flags().StringVar(&snapshotID, "snapshot", snapshotID, "Snapshot ID to be deleted")
	return cmd
}
