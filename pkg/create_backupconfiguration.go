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
	"strings"

	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"

	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/util/templates"
	kmapi "kmodules.xyz/client-go/api/v1"
)

var createBackupConfigExample = templates.Examples(`
		# Create a new BackupConfiguration
		# stash create backupconfig --namespace=<namespace> gcs-repo [Flag]
        # For Restic driver
        stash create backupconfig ss-backup --namespace=demo --repo-name=gcs-repo --schedule="*/4 * * * *" --target-apiversion=apps/v1 --target-kind=StatefulSet --target-name=stash-demo --paths=/source/data --volume-mounts=source-data:/source/data --keep-last=5 --prune=true
        # For VolumeSnapshotter driver
         stash create backupconfig statefulset-volume-snapshot --namespace=demo --driver=VolumeSnapshotter --schedule="*/4 * * * *" --target-apiversion=apps/v1 --target-kind=StatefulSet --target-name=stash-demo --replica=1 --volumesnpashotclass=default-snapshot-class --keep-last=5 --prune=true`)

type backupConfigOption struct {
	paths        []string
	volumeMounts []string

	targetRef           v1beta1.TargetRef
	retentionPolicy     v1alpha1.RetentionPolicy
	repository          kmapi.ObjectReference
	schedule            string
	driver              string
	volumesnpashotclass string
	task                string
	replica             int32
}

func NewCmdCreateBackupConfiguration() *cobra.Command {
	backupConfigOpt := backupConfigOption{}
	cmd := &cobra.Command{
		Use:               "backupconfig",
		Short:             `Create a new BackupConfiguration`,
		Long:              `Create a new BackupConfiguration to backup a target`,
		Example:           createBackupConfigExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("BackupConfiguration name is not provided")
			}

			backupConfigName := args[0]

			backupConfig, err := backupConfigOpt.newBackupConfiguration(backupConfigName, namespace)
			if err != nil {
				return err
			}
			_, err = createBackupConfiguration(backupConfig, backupConfig.ObjectMeta)
			if err != nil {
				return err
			}
			klog.Infof("BackupConfiguration %s/%s has been created successfully.", backupConfig.Namespace, backupConfig.Name)
			return err
		},
	}

	cmd.Flags().StringVar(&backupConfigOpt.targetRef.APIVersion, "target-apiversion", backupConfigOpt.targetRef.APIVersion, "API-Version of the target resource")
	cmd.Flags().StringVar(&backupConfigOpt.targetRef.Kind, "target-kind", backupConfigOpt.targetRef.Kind, "Kind of the target resource")
	cmd.Flags().StringVar(&backupConfigOpt.targetRef.Name, "target-name", backupConfigOpt.targetRef.Name, "Name of the target resource")
	cmd.Flags().StringVar(&backupConfigOpt.repository.Name, "repo-name", backupConfigOpt.repository.Name, "Name of the Repository")
	cmd.Flags().StringVar(&backupConfigOpt.repository.Namespace, "repo-namespace", namespace, "Namespace of the Repository")
	cmd.Flags().StringVar(&backupConfigOpt.schedule, "schedule", backupConfigOpt.schedule, "Schedule of the Backup")
	cmd.Flags().StringVar(&backupConfigOpt.driver, "driver", backupConfigOpt.driver, "Driver indicates the mechanism used to backup (i.e. VolumeSnapshotter, Restic)")
	cmd.Flags().StringVar(&backupConfigOpt.task, "task", backupConfigOpt.task, "Name of the Task")
	cmd.Flags().StringVar(&backupConfigOpt.volumesnpashotclass, "volumesnpashotclass", backupConfigOpt.volumesnpashotclass, "Name of the VolumeSnapshotClass")
	cmd.Flags().Int32Var(&backupConfigOpt.replica, "replica", backupConfigOpt.replica, "Replica specifies the number of replicas whose data should be backed up")

	cmd.Flags().Int64Var(&backupConfigOpt.retentionPolicy.KeepLast, "keep-last", backupConfigOpt.retentionPolicy.KeepLast, "Specify value for retention strategy")
	cmd.Flags().Int64Var(&backupConfigOpt.retentionPolicy.KeepHourly, "keep-hourly", backupConfigOpt.retentionPolicy.KeepHourly, "Specify value for retention strategy")
	cmd.Flags().Int64Var(&backupConfigOpt.retentionPolicy.KeepDaily, "keep-daily", backupConfigOpt.retentionPolicy.KeepDaily, "Specify value for retention strategy")
	cmd.Flags().Int64Var(&backupConfigOpt.retentionPolicy.KeepWeekly, "keep-weekly", backupConfigOpt.retentionPolicy.KeepWeekly, "Specify value for retention strategy")
	cmd.Flags().Int64Var(&backupConfigOpt.retentionPolicy.KeepMonthly, "keep-monthly", backupConfigOpt.retentionPolicy.KeepMonthly, "Specify value for retention strategy")
	cmd.Flags().Int64Var(&backupConfigOpt.retentionPolicy.KeepYearly, "keep-yearly", backupConfigOpt.retentionPolicy.KeepYearly, "Specify value for retention strategy")
	cmd.Flags().BoolVar(&backupConfigOpt.retentionPolicy.Prune, "prune", backupConfigOpt.retentionPolicy.Prune, "Specify whether to prune old snapshot data")
	cmd.Flags().BoolVar(&backupConfigOpt.retentionPolicy.DryRun, "dry-run", backupConfigOpt.retentionPolicy.DryRun, "Specify whether to test retention policy without deleting actual data")

	cmd.Flags().StringSliceVar(&backupConfigOpt.paths, "paths", backupConfigOpt.paths, "List of paths to backup")
	cmd.Flags().StringSliceVar(&backupConfigOpt.volumeMounts, "volume-mounts", backupConfigOpt.volumeMounts, "List of volumes and their mountPaths")

	return cmd
}

func (opt backupConfigOption) newBackupConfiguration(name string, namespace string) (*v1beta1.BackupConfiguration, error) {
	backupConfig := &v1beta1.BackupConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.BackupConfigurationSpec{
			Schedule:        opt.schedule,
			RetentionPolicy: getRetentionPolicy(opt),
		},
	}

	if v1beta1.Snapshotter(opt.driver) == v1beta1.VolumeSnapshotter {
		backupConfig.Spec.Driver = v1beta1.Snapshotter(opt.driver)
	} else {
		backupConfig.Spec.Repository = kmapi.ObjectReference{
			Name:      opt.repository.Name,
			Namespace: opt.repository.Namespace,
		}
		backupConfig.Spec.Task = v1beta1.TaskRef{Name: opt.task}
	}

	err := opt.setBackupTarget(backupConfig)
	if err != nil {
		return nil, err
	}
	return backupConfig, nil
}

func createBackupConfiguration(backupConfig *v1beta1.BackupConfiguration, meta metav1.ObjectMeta) (*v1beta1.BackupConfiguration, error) {
	backupConfig, _, err := v1beta1_util.CreateOrPatchBackupConfiguration(
		context.TODO(),
		stashClient.StashV1beta1(),
		meta,
		func(in *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
			in.Spec = backupConfig.Spec
			return in
		},
		metav1.PatchOptions{},
	)
	return backupConfig, err
}

func (opt backupConfigOption) setBackupTarget(backupConfig *v1beta1.BackupConfiguration) error {
	if v1beta1.Snapshotter(opt.driver) == v1beta1.VolumeSnapshotter {
		backupConfig.Spec.Target = &v1beta1.BackupTarget{
			Ref:                     opt.targetRef,
			VolumeSnapshotClassName: opt.volumesnpashotclass,
		}
	} else {
		backupConfig.Spec.Target = &v1beta1.BackupTarget{
			Ref:   opt.targetRef,
			Paths: opt.paths,
		}
		// Configure VolumeMounts
		volumeMounts, err := getVolumeMounts(opt.volumeMounts)
		if err != nil {
			return err
		}
		backupConfig.Spec.Target.VolumeMounts = volumeMounts
	}
	if opt.replica > 0 {
		backupConfig.Spec.Target.Replicas = &opt.replica
	}
	return nil
}

func getVolumeMounts(volumeMounts []string) ([]core.VolumeMount, error) {
	// extract volume and mount information
	// then return the volumeMounts of the target
	volMounts := make([]core.VolumeMount, 0)
	for _, m := range volumeMounts {
		vol := strings.Split(m, ":")
		if len(vol) == 3 {
			volMounts = append(volMounts, core.VolumeMount{Name: vol[0], MountPath: vol[1], SubPath: vol[2]})
		} else if len(vol) == 2 {
			volMounts = append(volMounts, core.VolumeMount{Name: vol[0], MountPath: vol[1]})
		} else {
			return volMounts, fmt.Errorf("invalid volume-mounts. use either 'volName:mountPath' or 'volName:mountPath:subPath' format")
		}
	}

	return volMounts, nil
}

func getRetentionPolicy(opt backupConfigOption) v1alpha1.RetentionPolicy {
	retentionPolicy := opt.retentionPolicy
	if retentionPolicy.KeepLast > 0 {
		retentionPolicy.Name = fmt.Sprintf("%s-%d", "keep-last", opt.retentionPolicy.KeepLast)
	}
	if retentionPolicy.KeepDaily > 0 {
		retentionPolicy.Name = fmt.Sprintf("%s-%d", "keep-daily", opt.retentionPolicy.KeepDaily)
	}
	if retentionPolicy.KeepHourly > 0 {
		retentionPolicy.Name = fmt.Sprintf("%s-%d", "keep-hourly", opt.retentionPolicy.KeepHourly)
	}
	if retentionPolicy.KeepWeekly > 0 {
		retentionPolicy.Name = fmt.Sprintf("%s-%d", "keep-weekly", opt.retentionPolicy.KeepWeekly)
	}
	if retentionPolicy.KeepMonthly > 0 {
		retentionPolicy.Name = fmt.Sprintf("%s-%d", "keep-monthly", opt.retentionPolicy.KeepMonthly)
	}
	if retentionPolicy.KeepYearly > 0 {
		retentionPolicy.Name = fmt.Sprintf("%s-%d", "keep-yearly", opt.retentionPolicy.KeepYearly)
	}
	return retentionPolicy
}
