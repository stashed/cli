package pkg

import (
	"context"
	"fmt"
	filepathx "gomodules.xyz/x/filepath"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
)

func getBackendMountingPod(kubeClient kubernetes.Interface, repo *v1alpha1.Repository) (*core.Pod, error) {
	vol, mnt := repo.Spec.Backend.Local.ToVolumeAndMount(repo.Name)
	var err error
	if repo.LocalNetworkVolume() {
		mnt.MountPath, err = filepathx.SecureJoin("/", repo.Name, mnt.MountPath, repo.LocalNetworkVolumePath())
		if err != nil {
			return nil, fmt.Errorf("failed to calculate filepath, reason: %s", err)
		}
	}
	// list all the pods
	podList, err := kubeClient.CoreV1().Pods(repo.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// return the pod that has the vol and mnt
	for i := range podList.Items {
		if hasVolume(podList.Items[i].Spec.Volumes, vol) {
			for _, c := range podList.Items[i].Spec.Containers {
				if hasVolumeMount(c.VolumeMounts, mnt) {
					return &podList.Items[i], nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no backend mounting pod found for Repository %v", repo.Name)
}

func hasVolume(volumes []core.Volume, vol core.Volume) bool {
	for i := range volumes {
		if volumes[i].Name == vol.Name {
			return true
		}
	}
	return false
}

func hasVolumeMount(mounts []core.VolumeMount, mnt core.VolumeMount) bool {
	for i := range mounts {
		if mounts[i].Name == mnt.Name && mounts[i].MountPath == mnt.MountPath {
			return true
		}
	}
	return false
}
