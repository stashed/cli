package pkg

import (
	"fmt"
	"github.com/evanphx/json-patch"
	"github.com/golang/glog"
	"github.com/json-iterator/go"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"kmodules.xyz/client-go"

	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	vs_v1alpha1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vs_api "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned/typed/volumesnapshot/v1alpha1"
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

			// get volumeSnapshot object
			vs, err := vsClient.VolumesnapshotV1alpha1().VolumeSnapshots(srcNamespace).Get(volumeSnapshotName, v1.GetOptions{})
			if err != nil {
				return err
			}

			// copy VolumeSnapshot to new namespace
			err = copyVolumeSnapshot(vs)
			if err != nil {
				return err
			}

			log.Infof("VolumeSnapshot %s/%s has been copied to %s namespace successfully.", vs.Namespace, vs.Name, dstNamespace)
			return nil
		},
	}

	return cmd
}

// Copy VolumeSnapshot
func copyVolumeSnapshot(vs *vs_v1alpha1.VolumeSnapshot) error{

	newObj := vs_v1alpha1.VolumeSnapshot{
		ObjectMeta : metav1.ObjectMeta{
			Name: vs.Name,
			Namespace: dstNamespace,
		},
		Spec: vs.Spec,
	}

	_, err := vsClient.VolumesnapshotV1alpha1().VolumeSnapshots(dstNamespace).Get(newObj.Name, v1.GetOptions{})
	if err != nil {
		_, err := vsClient.VolumesnapshotV1alpha1().VolumeSnapshots(dstNamespace).Create(&newObj)
		if err != nil {
			return err
		}
	}
	_, _, err = PatchVolumeSnapshot(vsClient.VolumesnapshotV1alpha1(), vs, func(obj *vs_v1alpha1.VolumeSnapshot) *vs_v1alpha1.VolumeSnapshot {
		obj.Spec = vs.Spec
		return obj
	},)

	return err
}

func PatchVolumeSnapshot(c vs_api.VolumesnapshotV1alpha1Interface, cur *vs_v1alpha1.VolumeSnapshot, transform func(alert *vs_v1alpha1.VolumeSnapshot) *vs_v1alpha1.VolumeSnapshot) (*vs_v1alpha1.VolumeSnapshot, kutil.VerbType, error) {
	return PatchVolumesnapshotObject(c, cur, transform(cur.DeepCopy()))
}

func PatchVolumesnapshotObject(c vs_api.VolumesnapshotV1alpha1Interface, cur, mod *vs_v1alpha1.VolumeSnapshot) (*vs_v1alpha1.VolumeSnapshot, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := jsonpatch.CreateMergePatch(curJson, modJson)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	glog.V(3).Infof("Patching Repository %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.VolumeSnapshots(cur.Namespace).Patch(cur.Name, types.MergePatchType, patch)
	return out, kutil.VerbPatched, err
}
