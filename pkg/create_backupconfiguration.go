package pkg

import (
	"fmt"
	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubectl/util/templates"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	v1beta1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"strings"
)

var (
	createBackupConfigExample = templates.Examples(`
		# Create a new BackupConfiguration
		# stash create backupconfig --namespace=<namespace> gcs-repo [Flag]
        # For Restic driver
        stash create backupconfig ss-backup --namespace=demo --repository=gcs-repo --schedule="*/4 * * * *" --target-apiversion=apps/v1 --target-kind=StatefulSet --target-name=stash-demo --paths=/source/data --volume-mounts=source-data:/source/data --keep-last=5 --prune=true
        # For VolumeSnapshotter driver
         stash create backupconfig statefulset-volume-snapshot --namespace=demo --driver=VolumeSnapshotter --schedule="*/4 * * * *" --target-apiversion=apps/v1 --target-kind=StatefulSet --target-name=stash-demo --replica=1 --volumesnpashotclass=default-snapshot-class --keep-last=5 --prune=true`)

	backupConfigOpt = backupConfigOption{}
)

type backupConfigOption struct {
	paths        []string
	volumeMounts []string

	targetRef           v1beta1.TargetRef
	retentionPolicy     v1alpha1.RetentionPolicy
	repository          string
	schedule            string
	driver              string
	volumesnpashotclass string
	task                string
	replica             int32
}

func NewCmdCreateBackupConfiguration() *cobra.Command {
	var cmd = &cobra.Command{
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

			backupConfig, err := createBackupConfiguration(backupConfigName)
			if err != nil {
				return err
			}
			log.Infof("BackupConfiguration %s/%s has been created successfully.", backupConfig.Namespace, backupConfig.Name)
			return err

		},
	}

	cmd.Flags().StringVar(&backupConfigOpt.targetRef.APIVersion, "target-apiversion", backupConfigOpt.targetRef.APIVersion, "API-Version of the target resource")
	cmd.Flags().StringVar(&backupConfigOpt.targetRef.Kind, "target-kind", backupConfigOpt.targetRef.Kind, "Kind of the target resource")
	cmd.Flags().StringVar(&backupConfigOpt.targetRef.Name, "target-name", backupConfigOpt.targetRef.Name, "Name of the target resource")
	cmd.Flags().StringVar(&backupConfigOpt.repository, "repository", backupConfigOpt.repository, "Name of the Repository")
	cmd.Flags().StringVar(&backupConfigOpt.schedule, "schedule", backupConfigOpt.schedule, "Schedule of the Backup")
	cmd.Flags().StringVar(&backupConfigOpt.driver, "driver", backupConfigOpt.driver, "Driver indicates the mechanism used to backup (i.e. VolumeSnapshotter, Restic)")
	cmd.Flags().StringVar(&backupConfigOpt.task, "task", backupConfigOpt.task, "Name of the Task")
	cmd.Flags().StringVar(&backupConfigOpt.volumesnpashotclass, "volumesnpashotclass", backupConfigOpt.volumesnpashotclass, "Name of the VolumeSnapshotClass")
	cmd.Flags().Int32Var(&backupConfigOpt.replica, "replica", backupConfigOpt.replica, "Replica specifies the number of replicas whose data should be backed up")

	cmd.Flags().IntVar(&backupConfigOpt.retentionPolicy.KeepLast, "keep-last", backupConfigOpt.retentionPolicy.KeepLast, "Specify value for retention strategy")
	cmd.Flags().IntVar(&backupConfigOpt.retentionPolicy.KeepHourly, "keep-hourly", backupConfigOpt.retentionPolicy.KeepHourly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&backupConfigOpt.retentionPolicy.KeepDaily, "keep-daily", backupConfigOpt.retentionPolicy.KeepDaily, "Specify value for retention strategy")
	cmd.Flags().IntVar(&backupConfigOpt.retentionPolicy.KeepWeekly, "keep-weekly", backupConfigOpt.retentionPolicy.KeepWeekly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&backupConfigOpt.retentionPolicy.KeepMonthly, "keep-monthly", backupConfigOpt.retentionPolicy.KeepMonthly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&backupConfigOpt.retentionPolicy.KeepYearly, "keep-yearly", backupConfigOpt.retentionPolicy.KeepYearly, "Specify value for retention strategy")
	cmd.Flags().BoolVar(&backupConfigOpt.retentionPolicy.Prune, "prune", backupConfigOpt.retentionPolicy.Prune, "Specify whether to prune old snapshot data")
	cmd.Flags().BoolVar(&backupConfigOpt.retentionPolicy.DryRun, "dry-run", backupConfigOpt.retentionPolicy.DryRun, "Specify whether to test retention policy without deleting actual data")

	cmd.Flags().StringSliceVar(&backupConfigOpt.paths, "paths", backupConfigOpt.paths, "List of paths to backup")
	cmd.Flags().StringSliceVar(&backupConfigOpt.volumeMounts, "volume-mounts", backupConfigOpt.volumeMounts, "List of volumes and their mountPaths")

	return cmd
}

func createBackupConfiguration(name string) (backupConfig *v1beta1.BackupConfiguration, err error) {

	backupConfig = &v1beta1.BackupConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.BackupConfigurationSpec{
			Schedule:        backupConfigOpt.schedule,
			RetentionPolicy: getRetentionPolicy(),
		},
	}

	if v1beta1.Snapshotter(backupConfigOpt.driver) == v1beta1.VolumeSnapshotter {
		backupConfig.Spec.Driver = v1beta1.Snapshotter(backupConfigOpt.driver)
	} else {
		backupConfig.Spec.Repository = core.LocalObjectReference{Name: backupConfigOpt.repository}
		backupConfig.Spec.Task = v1beta1.TaskRef{Name: backupConfigOpt.task}
	}

	err = setBackupTarget(backupConfig)
	if err != nil {
		return nil, err
	}

	backupConfig, _, err = v1beta1_util.CreateOrPatchBackupConfiguration(stashClient.StashV1beta1(), backupConfig.ObjectMeta, func(in *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
		in.Spec = backupConfig.Spec
		return in
	})
	return backupConfig, err

}

func setVolumeMounts(target interface{}) error {
	// extract volume and mount information
	// then configure the volumeMounts of the target
	volMounts := make([]core.VolumeMount, 0)
	for _, m := range backupConfigOpt.volumeMounts {
		vol := strings.Split(m, ":")
		if len(vol) == 3 {
			volMounts = append(volMounts, core.VolumeMount{Name: vol[0], MountPath: vol[1], SubPath: vol[2]})
		} else if len(vol) == 2 {
			volMounts = append(volMounts, core.VolumeMount{Name: vol[0], MountPath: vol[1]})
		} else {
			return fmt.Errorf("invalid volume-mounts. use either 'volName:mountPath' or 'volName:mountPath:subPath' format")
		}
	}

	switch target.(type) {
	case *v1beta1.BackupTarget:
		t := target.(*v1beta1.BackupTarget)
		t.VolumeMounts = volMounts
	case *v1beta1.RestoreTarget:
		t := target.(*v1beta1.RestoreTarget)
		t.VolumeMounts = volMounts
	}
	return nil
}

func setBackupTarget(backupConfig *v1beta1.BackupConfiguration) error {

	if v1beta1.Snapshotter(backupConfigOpt.driver) == v1beta1.VolumeSnapshotter {
		backupConfig.Spec.Target = &v1beta1.BackupTarget{
			Ref:                     backupConfigOpt.targetRef,
			VolumeSnapshotClassName: backupConfigOpt.volumesnpashotclass,
		}
		if backupConfigOpt.replica > 0 {
			backupConfig.Spec.Target.Replicas = &backupConfigOpt.replica
		}

	} else {
		backupConfig.Spec.Target = &v1beta1.BackupTarget{
			Ref:   backupConfigOpt.targetRef,
			Paths: backupConfigOpt.paths,
		}
		// Configure VolumeMounts
		err := setVolumeMounts(backupConfig.Spec.Target)
		if err != nil {
			return err
		}
	}
	return nil
}


func getRetentionPolicy() v1alpha1.RetentionPolicy{
	retentionPolicy := backupConfigOpt.retentionPolicy
	if retentionPolicy.KeepLast > 0 {
		retentionPolicy.Name = fmt.Sprintf("%s-%d","keep-last", backupConfigOpt.retentionPolicy.KeepLast)
	}
	if retentionPolicy.KeepDaily > 0 {
		retentionPolicy.Name = fmt.Sprintf("%s-%d","keep-daily", backupConfigOpt.retentionPolicy.KeepDaily)
	}
	if retentionPolicy.KeepHourly > 0 {
		retentionPolicy.Name = fmt.Sprintf("%s-%d","keep-hourly", backupConfigOpt.retentionPolicy.KeepHourly)
	}
	if retentionPolicy.KeepWeekly > 0 {
		retentionPolicy.Name = fmt.Sprintf("%s-%d","keep-weekly", backupConfigOpt.retentionPolicy.KeepWeekly)
	}
	if retentionPolicy.KeepMonthly > 0 {
		retentionPolicy.Name = fmt.Sprintf("%s-%d","keep-monthly", backupConfigOpt.retentionPolicy.KeepMonthly)
	}
	if retentionPolicy.KeepYearly > 0 {
		retentionPolicy.Name = fmt.Sprintf("%s-%d","keep-yearly", backupConfigOpt.retentionPolicy.KeepYearly)
	}
	return retentionPolicy
}