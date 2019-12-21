/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package pkg

import (
	"fmt"

	"github.com/appscode/go/log"
	jsoniter "github.com/json-iterator/go"
	vs_api "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1beta1"
	vs_v1alpha1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1beta1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var json = jsoniter.ConfigFastest

func NewCmdCopyVolumeSnapshot() *cobra.Command {
	var cmd = &cobra.Command{
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
			vs, err := vsClient.SnapshotV1beta1().VolumeSnapshots(srcNamespace).Get(volumeSnapshotName, metav1.GetOptions{})
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

			log.Infof("VolumeSnapshot %s/%s has been copied to %s namespace successfully.", vs.Namespace, vs.Name, dstNamespace)
			return nil
		},
	}

	return cmd
}

func createVolumeSnapshot(vs *vs_v1alpha1.VolumeSnapshot, meta metav1.ObjectMeta) (*vs_v1alpha1.VolumeSnapshot, error) {
	vs, _, err := CreateOrPatchVolumeSnapshot(vsClient.SnapshotV1beta1(), meta, func(in *vs_api.VolumeSnapshot) *vs_api.VolumeSnapshot {
		in.Spec = vs.Spec
		return in
	})
	return vs, err
}
