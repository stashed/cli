package pkg

import (
	"fmt"
	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubectl/util/templates"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	v1beta1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"strings"
)

var (
	createBackupConfigLong = templates.LongDesc(`
		Create a new BackupConfiguration`)

	createBackupConfigExample = templates.Examples(`
		# Create a new BackupConfiguration
		# stash create backupconfig --namespace=<namespace> gcs-repo [Flag]
        # For Restic driver
        stash create backupconfig ss-backup --namespace=demo --repository=gcs-repo --schedule="*/4 * * * *" --target-apiversion=apps/v1 --target-kind=StatefulSet --target-name=stash-demo --paths=/source/data --volume-mounts=source-data:/source/data --retention-name=keep-last-5 --retention-keep-last=5 --retention-prune=true
        # For VolumeSnapshotter driver
         stash create backupconfig statefulset-volume-snapshot --namespace=demo --driver=VolumeSnapshotter --schedule="*/4 * * * *" --target-apiversion=apps/v1 --target-kind=StatefulSet --target-name=stash-demo --replica=1 --vs-class-name=default-snapshot-class --retention-name=keep-last-5 --retention-keep-last=5 --retention-prune=true`)
)

func NewCmdCreateBackupConfiguration() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "backupconfig",
		Short:             `Create a repository`,
		Long:              createBackupConfigLong,
		Example:           createBackupConfigExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("BackupConfiguration name has not provided")
			}

			backupConfigName := args[0]

			backupConfig, err := createBackupConfiguration(backupConfigName)
			if err != nil {
				return err
			}
			log.Infof("BackupConfiguration %s/%s has been created successfully.", backupConfig.Namespace, backupConfig.Name)
			return err

		},
	}

	cmd.Flags().StringVar(&opt.TargetRef.APIVersion, "target-apiversion", opt.TargetRef.APIVersion, "API-Version of the target resource")
	cmd.Flags().StringVar(&opt.TargetRef.Kind, "target-kind", opt.TargetRef.Kind, "Kind of the target resource")
	cmd.Flags().StringVar(&opt.TargetRef.Name, "target-name", opt.TargetRef.Name, "Name of the target resource")
	cmd.Flags().StringVar(&opt.RepositoryName, "repository", opt.RepositoryName, "Name of the Repository")
	cmd.Flags().StringVar(&opt.Schedule, "schedule", opt.Schedule, "Schedule of the Backup")
	cmd.Flags().StringVar(&opt.Driver, "driver", opt.Driver, "Driver indicates the mechanism used to backup (i.e. VolumeSnapshotter, Restic)")
	cmd.Flags().StringVar(&opt.TaskName, "task", opt.TaskName, "Name of the Task")
	cmd.Flags().StringVar(&opt.VSClassName, "vs-class-name", opt.VSClassName, "Name of the VolumeSnapshotClass")
	cmd.Flags().Int32Var(&opt.Replica, "replica", opt.Replica, "Replica specifies the number of replicas whose data should be backed up")
	cmd.Flags().BoolVar(&opt.Pause, "pause", opt.Pause, "Pause enable/disable switch for backup")

	cmd.Flags().StringVar(&opt.RetentionPolicy.Name, "retention-name", opt.RetentionPolicy.Name, "Specify name for retention strategy")
	cmd.Flags().IntVar(&opt.RetentionPolicy.KeepLast, "retention-keep-last", opt.RetentionPolicy.KeepLast, "Specify value for retention strategy")
	cmd.Flags().IntVar(&opt.RetentionPolicy.KeepHourly, "retention-keep-hourly", opt.RetentionPolicy.KeepHourly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&opt.RetentionPolicy.KeepDaily, "retention-keep-daily", opt.RetentionPolicy.KeepDaily, "Specify value for retention strategy")
	cmd.Flags().IntVar(&opt.RetentionPolicy.KeepWeekly, "retention-keep-weekly", opt.RetentionPolicy.KeepWeekly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&opt.RetentionPolicy.KeepMonthly, "retention-keep-monthly", opt.RetentionPolicy.KeepMonthly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&opt.RetentionPolicy.KeepYearly, "retention-keep-yearly", opt.RetentionPolicy.KeepYearly, "Specify value for retention strategy")
	cmd.Flags().StringSliceVar(&opt.RetentionPolicy.KeepTags, "retention-keep-tags", opt.RetentionPolicy.KeepTags, "Specify value for retention strategy")
	cmd.Flags().BoolVar(&opt.RetentionPolicy.Prune, "retention-prune", opt.RetentionPolicy.Prune, "Specify whether to prune old snapshot data")
	cmd.Flags().BoolVar(&opt.RetentionPolicy.DryRun, "retention-dry-run", opt.RetentionPolicy.DryRun, "Specify whether to test retention policy without deleting actual data")

	cmd.Flags().StringVar(&opt.Paths, "paths", opt.Paths, "List of paths to backup")
	cmd.Flags().StringVar(&opt.VolMounts, "volume-mounts", opt.VolMounts, "List of volumes and their mountPaths")

	return cmd
}

func createBackupConfiguration(name string) (backupConfig *v1beta1.BackupConfiguration, err error) {
	if v1beta1.Snapshotter(opt.Driver) != v1beta1.VolumeSnapshotter {
		// Configure VolumeMounts and Backup Paths
		err = configureVolumeMountsAndPathsOrSnapshots()
		if err != nil {
			return backupConfig, err
		}
	}

	backupConfig = getBackupConfigurationObj(name)

	backupConfig, _, err = v1beta1_util.CreateOrPatchBackupConfiguration(stashClient.StashV1beta1(),
		metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		func(obj *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
			obj.Spec = backupConfig.Spec
			return obj
		})
	return backupConfig, err

}

func configureVolumeMountsAndPathsOrSnapshots() error {
	// extract volume and mount information
	mounts := strings.Split(opt.VolMounts, ",")
	for _, m := range mounts {
		vol := strings.Split(m, ":")
		if len(vol) == 3 {
			volumeMounts = append(volumeMounts, core.VolumeMount{Name: vol[0], MountPath: vol[1], SubPath: vol[2]})
		} else if len(vol) == 2 {
			volumeMounts = append(volumeMounts, core.VolumeMount{Name: vol[0], MountPath: vol[1]})
		} else {
			return fmt.Errorf("invalid volume-mounts. use either 'volName:mountPath' or 'volName:mountPath:subPath' format")
		}
	}
	// extract paths information
	paths = strings.Split(opt.Paths, ",")
	if len(paths) < 1 {
		return fmt.Errorf("Paths have not defined properly ")
	}
	//extract Snapshot information
	snapshots = strings.Split(opt.Rule.Snapshot, ",")
	if len(snapshots) < 1 {
		return fmt.Errorf("Snapshot have not defined properly ")
	}
	return nil
}

func getBackupConfigurationObj(name string) *v1beta1.BackupConfiguration {
	backupConfig := &v1beta1.BackupConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1beta1.ResourceKindBackupConfiguration,
			APIVersion: v1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.BackupConfigurationSpec{
			Schedule: opt.Schedule,
			Target: &v1beta1.BackupTarget{
				Ref: opt.TargetRef,
			},
			RetentionPolicy: opt.RetentionPolicy,
			Paused:          opt.Pause,
		},
	}
	if v1beta1.Snapshotter(opt.Driver) == v1beta1.VolumeSnapshotter {
		backupConfig.Spec.Driver = v1beta1.Snapshotter(opt.Driver)
		backupConfig.Spec.Target.VolumeSnapshotClassName = opt.VSClassName
		backupConfig.Spec.Target.Replicas = &opt.Replica

	} else {
		backupConfig.Spec.Repository = core.LocalObjectReference{Name: opt.RepositoryName}
		backupConfig.Spec.Target.Paths = paths
		backupConfig.Spec.Target.VolumeMounts = volumeMounts
		backupConfig.Spec.Task = v1beta1.TaskRef{Name: opt.TaskName}
	}
	return backupConfig
}
