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

	vsapi "github.com/kubernetes-csi/external-snapshotter/client/v7/apis/volumesnapshot/v1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	vsu "kmodules.xyz/csi-utils/volumesnapshot/v1"
)

func NewCmdCopyVolumeSnapshot() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "volumesnapshot",
		Short:             `Copy VolumeSnapshot`,
		Long:              `Copy VolumeSnapshot from one namespace to another namespace`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("volumeSnapshot name not found")
			}

			volumeSnapshotName := args[0]

			// get source VolumeSnapshot object
			vs, err := vsClient.SnapshotV1().VolumeSnapshots(srcNamespace).Get(context.TODO(), volumeSnapshotName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// copy the VolumeSnapshot to new namespace
			meta := metav1.ObjectMeta{
				Name:        vs.Name,
				Namespace:   dstNamespace,
				Labels:      vs.Labels,
				Annotations: vs.Annotations,
			}
			vs, err = createVolumeSnapshot(vs, meta)
			if err != nil {
				return err
			}

			klog.Infof("VolumeSnapshot %s/%s has been copied to %s namespace successfully.", vs.Namespace, vs.Name, dstNamespace)
			return nil
		},
	}

	return cmd
}

func createVolumeSnapshot(vs *vsapi.VolumeSnapshot, meta metav1.ObjectMeta) (*vsapi.VolumeSnapshot, error) {
	vs, _, err := vsu.CreateOrPatchVolumeSnapshot(context.TODO(), vsClient, meta, func(in *vsapi.VolumeSnapshot) *vsapi.VolumeSnapshot {
		in.Spec = vs.Spec
		return in
	}, metav1.PatchOptions{})
	return vs, err
}
