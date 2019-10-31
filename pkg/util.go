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
	"time"

	"stash.appscode.dev/stash/apis/stash/v1beta1"

	"github.com/evanphx/json-patch"
	"github.com/golang/glog"
	vs_api "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	vs "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned/typed/volumesnapshot/v1alpha1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd/api"
	kutil "kmodules.xyz/client-go"
)

const (
	PullInterval = time.Second * 2
	WaitTimeOut  = time.Minute * 10
)

func CreateOrPatchVolumeSnapshot(c vs.SnapshotV1alpha1Interface, meta metav1.ObjectMeta, transform func(alert *vs_api.VolumeSnapshot) *vs_api.VolumeSnapshot) (*vs_api.VolumeSnapshot, kutil.VerbType, error) {
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

func PatchVolumeSnapshot(c vs.SnapshotV1alpha1Interface, cur *vs_api.VolumeSnapshot, transform func(alert *vs_api.VolumeSnapshot) *vs_api.VolumeSnapshot) (*vs_api.VolumeSnapshot, kutil.VerbType, error) {
	return PatchVolumesnapshotObject(c, cur, transform(cur.DeepCopy()))
}

func PatchVolumesnapshotObject(c vs.SnapshotV1alpha1Interface, cur, mod *vs_api.VolumeSnapshot) (*vs_api.VolumeSnapshot, kutil.VerbType, error) {
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

func WaitUntilBackupSessionCompleted(name string, namespace string) error {
	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		backupSession, err := stashClient.StashV1beta1().BackupSessions(namespace).Get(name, metav1.GetOptions{})
		if err == nil {
			if backupSession.Status.Phase == v1beta1.BackupSessionSucceeded {
				return true, nil
			}
			if backupSession.Status.Phase == v1beta1.BackupSessionFailed {
				return true, fmt.Errorf("BackupSession has been failed")
			}
		}
		return false, nil
	})
}

func WaitUntilRestoreSessionCompleted(name string, namespace string) error {
	return wait.PollImmediate(PullInterval, WaitTimeOut, func() (done bool, err error) {
		restoreSession, err := stashClient.StashV1beta1().RestoreSessions(namespace).Get(name, metav1.GetOptions{})
		if err == nil {
			if restoreSession.Status.Phase == v1beta1.RestoreSessionSucceeded {
				return true, nil
			}
			if restoreSession.Status.Phase == v1beta1.RestoreSessionFailed {
				return true, fmt.Errorf("RestoreSession has been failed")
			}
		}
		return false, nil
	})
}
