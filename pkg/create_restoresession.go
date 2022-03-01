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
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"

	vs "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1beta1"
	"github.com/spf13/cobra"
	"gomodules.xyz/pointer"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/util/templates"
	kmapi "kmodules.xyz/client-go/api/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

var createRestoreSessionExample = templates.Examples(`
		# Create a RestoreSession
		# stash create restore --namespace=demo <restore session name> [Flag]
        # For Restic driver
         stash create restoresession ss-restore --namespace=demo --repo-name=gcs-repo --target-apiversion=apps/v1 --target-kind=StatefulSet --target-name=stash-recovered --paths=/source/data --volume-mounts=source-data:/source/data
        # For VolumeSnapshotter driver
         stash create restoresession restore-pvc --namespace=demo --driver=VolumeSnapshotter --replica=3 --claim.name=restore-data-restore-demo-${POD_ORDINAL} --claim.access-modes=ReadWriteOnce --claim.storageclass=standard --claim.size=1Gi --claim.datasource=source-data-stash-demo-0-1567146010`)

type restoreSessionOption struct {
	volumeMounts []string
	task         string
	targetRef    v1beta1.TargetRef
	repository   kmapi.ObjectReference
	driver       string
	replica      int32
	alias        string

	rule                v1beta1.Rule
	volumeClaimTemplate volumeclaimTemplate
}

type volumeclaimTemplate struct {
	name         string
	accessModes  []string
	storageClass string
	size         string
	dataSource   string
}

func NewCmdCreateRestoreSession() *cobra.Command {
	restoreSessionOpt := restoreSessionOption{}
	cmd := &cobra.Command{
		Use:               "restoresession",
		Short:             `Create a new RestoreSession`,
		Long:              `Create a new RestoreSession to restore backed up data`,
		Example:           createRestoreSessionExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("RestoreSession name is not provided")
			}

			restoresessionName := args[0]

			restoreSession, err := restoreSessionOpt.newRestoreSession(restoresessionName, namespace)
			if err != nil {
				return nil
			}

			restoreSession, err = createRestoreSession(restoreSession)
			if err != nil {
				return err
			}
			klog.Infof("RestoreSession %s/%s has been created successfully.", restoreSession.Namespace, restoreSession.Name)
			return err
		},
	}

	cmd.Flags().StringVar(&restoreSessionOpt.targetRef.APIVersion, "target-apiversion", restoreSessionOpt.targetRef.APIVersion, "API-Version of the target resource")
	cmd.Flags().StringVar(&restoreSessionOpt.targetRef.Kind, "target-kind", restoreSessionOpt.targetRef.Kind, "Kind of the target resource")
	cmd.Flags().StringVar(&restoreSessionOpt.targetRef.Name, "target-name", restoreSessionOpt.targetRef.Name, "Name of the target resource")

	cmd.Flags().StringVar(&restoreSessionOpt.repository.Name, "repo-name", restoreSessionOpt.repository.Name, "Name of the Repository")
	cmd.Flags().StringVar(&restoreSessionOpt.repository.Namespace, "repo-namespace", namespace, "Namespace of the Repository")
	cmd.Flags().StringVar(&restoreSessionOpt.driver, "driver", restoreSessionOpt.driver, "Driver indicates the mechanism used to backup (i.e. VolumeSnapshotter, Restic)")
	cmd.Flags().StringVar(&restoreSessionOpt.task, "task", restoreSessionOpt.task, "Name of the Task")
	cmd.Flags().Int32Var(&restoreSessionOpt.replica, "replica", restoreSessionOpt.replica, "Replica specifies the number of replicas whose data should be backed up")
	cmd.Flags().StringVar(&restoreSessionOpt.alias, "alias", restoreSessionOpt.alias, "Host identifier of the backed up data. It must be same as the alias used during backup")

	cmd.Flags().StringSliceVar(&restoreSessionOpt.volumeMounts, "volume-mounts", restoreSessionOpt.volumeMounts, "List of volumes and their mountPaths")
	cmd.Flags().StringSliceVar(&restoreSessionOpt.rule.Paths, "paths", restoreSessionOpt.rule.Paths, "List of paths to backup")
	cmd.Flags().StringSliceVar(&restoreSessionOpt.rule.Snapshots, "snapshots", restoreSessionOpt.rule.Snapshots, "Name of the Snapshot(single)")
	cmd.Flags().StringVar(&restoreSessionOpt.rule.SourceHost, "host", restoreSessionOpt.rule.SourceHost, "Name of the Source host")
	cmd.Flags().StringVar(&restoreSessionOpt.volumeClaimTemplate.name, "claim.name", restoreSessionOpt.volumeClaimTemplate.name, "Name of the VolumeClaimTemplate")
	cmd.Flags().StringSliceVar(&restoreSessionOpt.volumeClaimTemplate.accessModes, "claim.access-modes", restoreSessionOpt.volumeClaimTemplate.accessModes, "Access mode of the VolumeClaimTemplates")
	cmd.Flags().StringVar(&restoreSessionOpt.volumeClaimTemplate.storageClass, "claim.storageclass", restoreSessionOpt.volumeClaimTemplate.storageClass, "Name of the Storage secret for VolumeClaimTemplate")
	cmd.Flags().StringVar(&restoreSessionOpt.volumeClaimTemplate.size, "claim.size", restoreSessionOpt.volumeClaimTemplate.size, "Total requested size of the VolumeClaimTemplate")
	cmd.Flags().StringVar(&restoreSessionOpt.volumeClaimTemplate.dataSource, "claim.datasource", restoreSessionOpt.volumeClaimTemplate.dataSource, "DataSource of the VolumeClaimTemplate")

	return cmd
}

func (opt restoreSessionOption) newRestoreSession(name string, namespace string) (*v1beta1.RestoreSession, error) {
	restoreSession := &v1beta1.RestoreSession{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if v1beta1.Snapshotter(opt.driver) == v1beta1.VolumeSnapshotter {
		restoreSession.Spec.Driver = v1beta1.Snapshotter(opt.driver)
	} else {
		restoreSession.Spec = v1beta1.RestoreSessionSpec{
			Repository: kmapi.ObjectReference{
				Name:      opt.repository.Name,
				Namespace: opt.repository.Namespace,
			},
		}
		restoreSession.Spec.Task = v1beta1.TaskRef{Name: opt.task}
	}
	err := opt.setRestoreTarget(restoreSession)
	if err != nil {
		return nil, err
	}
	return restoreSession, nil
}

func createRestoreSession(restoreSession *v1beta1.RestoreSession) (*v1beta1.RestoreSession, error) {
	restoreSession, _, err := v1beta1_util.CreateOrPatchRestoreSession(
		context.TODO(),
		stashClient.StashV1beta1(),
		restoreSession.ObjectMeta,
		func(in *v1beta1.RestoreSession) *v1beta1.RestoreSession {
			in.Spec = restoreSession.Spec
			return in
		},
		metav1.PatchOptions{},
	)
	return restoreSession, err
}

func (opt restoreSessionOption) setRestoreTarget(restoreSession *v1beta1.RestoreSession) error {
	// if driver is VolumeSnapshotter then configure the VolumeClaimTemplates
	// otherwise configure the TargetRef for sidecar model or configure the volumeClaimTemplates for cronJob model
	if v1beta1.Snapshotter(opt.driver) == v1beta1.VolumeSnapshotter {
		restoreSession.Spec.Target = &v1beta1.RestoreTarget{
			VolumeClaimTemplates: opt.getRestoredPVCTemplates(),
		}
		restoreSession.Spec.Target.VolumeClaimTemplates[0].Spec.DataSource = &core.TypedLocalObjectReference{
			Kind:     "VolumeSnapshot",
			Name:     opt.volumeClaimTemplate.dataSource,
			APIGroup: pointer.StringP(vs.GroupName),
		}
	} else {
		if opt.volumeClaimTemplate.name != "" && opt.volumeClaimTemplate.size != "" {
			restoreSession.Spec.Target = &v1beta1.RestoreTarget{
				VolumeClaimTemplates: opt.getRestoredPVCTemplates(),
			}
		} else {
			restoreSession.Spec.Target = &v1beta1.RestoreTarget{
				Ref: opt.targetRef,
			}
		}

		if len(opt.volumeMounts) > 0 {
			volumeMounts, err := getVolumeMounts(opt.volumeMounts)
			if err != nil {
				return err
			}
			restoreSession.Spec.Target.VolumeMounts = volumeMounts
		}
		restoreSession.Spec.Target.Alias = opt.alias
		restoreSession.Spec.Target.Rules = []v1beta1.Rule{opt.rule}
	}
	if opt.replica > 0 {
		restoreSession.Spec.Target.Replicas = &opt.replica
	}
	return nil
}

func (opt restoreSessionOption) getRestoredPVCTemplates() []ofst.PersistentVolumeClaim {
	pvcs := []ofst.PersistentVolumeClaim{
		{
			PartialObjectMeta: ofst.PartialObjectMeta{
				Name:      opt.volumeClaimTemplate.name,
				Namespace: namespace,
			},
			Spec: core.PersistentVolumeClaimSpec{
				AccessModes:      getPVAccessModes(opt.volumeClaimTemplate.accessModes),
				StorageClassName: &opt.volumeClaimTemplate.storageClass,
			},
		},
	}
	if opt.volumeClaimTemplate.size != "" {
		pvcs[0].Spec.Resources.Requests = core.ResourceList{
			core.ResourceStorage: resource.MustParse(opt.volumeClaimTemplate.size),
		}
	}
	return pvcs
}

func getPVAccessModes(acModes []string) []core.PersistentVolumeAccessMode {
	accessModes := make([]core.PersistentVolumeAccessMode, 0)
	for _, am := range acModes {
		accessModes = append(accessModes, core.PersistentVolumeAccessMode(am))
	}
	return accessModes
}
