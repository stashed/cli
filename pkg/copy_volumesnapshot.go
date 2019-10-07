package pkg

import (
	"fmt"

	"github.com/appscode/go/log"
	vs_api "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	vs_v1alpha1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
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
			vs, err := vsClient.VolumesnapshotV1alpha1().VolumeSnapshots(srcNamespace).Get(volumeSnapshotName, v1.GetOptions{})
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
	vs, _, err := CreateOrPatchVolumeSnapshot(vsClient.VolumesnapshotV1alpha1(), meta, func(in *vs_api.VolumeSnapshot) *vs_api.VolumeSnapshot {
		in.Spec = vs.Spec
		return in
	})
	return vs, err
}
