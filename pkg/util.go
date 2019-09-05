package pkg

import (
	"github.com/evanphx/json-patch"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd/api"
	"kmodules.xyz/client-go"
	vs_api "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vs "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned/typed/volumesnapshot/v1alpha1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"time"
)

const (
	PullInterval = time.Second * 2
	WaitTimeOut  = time.Minute * 10
)

func CreateOrPatchVolumeSnapshot(c vs.VolumesnapshotV1alpha1Interface, meta metav1.ObjectMeta, transform func(alert *vs_api.VolumeSnapshot) *vs_api.VolumeSnapshot) (*vs_api.VolumeSnapshot, kutil.VerbType, error) {
	cur, err := c.VolumeSnapshots(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating VolumeSnapshot %s/%s.", meta.Namespace, meta.Name)
		out, err := c.VolumeSnapshots(meta.Namespace).Create(transform(&vs_api.VolumeSnapshot{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VolumeSnapshot",
				APIVersion: api.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchVolumeSnapshot(c, cur, transform)
}

func PatchVolumeSnapshot(c vs.VolumesnapshotV1alpha1Interface, cur *vs_api.VolumeSnapshot, transform func(alert *vs_api.VolumeSnapshot) *vs_api.VolumeSnapshot) (*vs_api.VolumeSnapshot, kutil.VerbType, error) {
	return PatchVolumesnapshotObject(c, cur, transform(cur.DeepCopy()))
}

func PatchVolumesnapshotObject(c vs.VolumesnapshotV1alpha1Interface, cur, mod *vs_api.VolumeSnapshot) (*vs_api.VolumeSnapshot, kutil.VerbType, error) {
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
	glog.V(3).Infof("Patching VolumeSnapshot %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.VolumeSnapshots(cur.Namespace).Patch(cur.Name, types.MergePatchType, patch)
	return out, kutil.VerbPatched, err
}

func WaitUntilBackupSessionSucceed(name string, namespace string) error {
	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		backupSessionList, err := stashClient.StashV1beta1().BackupSessions(namespace).List(metav1.ListOptions{LabelSelector: fmt.Sprintf("stash.appscode.com/backup-configuration=%s", name)})
		if err == nil {
			for _, backupSession := range backupSessionList.Items {
				if backupSession.Status.Phase == v1beta1.BackupSessionSucceeded {
					return true, nil
				}

			}
		}
		return false, nil
	})
}

func WaitUntilRestoreSessionSucceed(name string, namespace string) error {
	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		restoreSession, err := stashClient.StashV1beta1().RestoreSessions(namespace).Get(name, metav1.GetOptions{})
		if err == nil {
			if restoreSession.Status.Phase == v1beta1.RestoreSessionSucceeded {
				return true, nil
			}
		}
		return false, nil
	})
}

func quantityTypePointer(q resource.Quantity) *resource.Quantity {
	return &q
}
