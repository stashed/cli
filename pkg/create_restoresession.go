package pkg

import (
	"fmt"
	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	vs "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubectl/util/templates"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	v1beta1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"strings"
	"time"
)

var (
	createRestoreSessionExample = templates.Examples(`
		# Create a RestoreSession
		# stash create restore --namespace=demo <restore session name> [Flag]
        # For Restic driver
         stash create restoresession ss-restore --namespace=demo --repository=gcs-repo --target-apiversion=apps/v1 --target-kind=StatefulSet --target-name=stash-recovered --paths=/source/data --volume-mounts=source-data:/source/data
        # For VolumeSnapshotter driver
         stash create restoresession restore-pvc --namespace=demo --driver=VolumeSnapshotter --replica=3 --claim.name=restore-data-restore-demo-${POD_ORDINAL} --claim.access-modes=ReadWriteOnce --claim.storageclass=standard --claim.size=1Gi --claim.datasource=source-data-stash-demo-0-1567146010`)

	restoreSessionOpt = restoreSessionOption{}
)

type restoreSessionOption struct {
	volumeMounts []string

	targetRef           v1beta1.TargetRef
	retentionPolicy     v1alpha1.RetentionPolicy
	repository          string
	schedule            string
	driver              string
	volumesnpashotclass string
	task                string
	replica             int32

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
	var cmd = &cobra.Command{
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

			restoreSession, err := createRestoreSession(restoresessionName, namespace)
			if err != nil {
				return err
			}
			log.Infof("RestoreSession %s/%s has been created successfully.", restoreSession.Namespace, restoreSession.Name)
			return err

		},
	}

	cmd.Flags().StringVar(&restoreSessionOpt.targetRef.APIVersion, "target-apiversion", restoreSessionOpt.targetRef.APIVersion, "API-Version of the target resource")
	cmd.Flags().StringVar(&restoreSessionOpt.targetRef.Kind, "target-kind", restoreSessionOpt.targetRef.Kind, "Kind of the target resource")
	cmd.Flags().StringVar(&restoreSessionOpt.targetRef.Name, "target-name", restoreSessionOpt.targetRef.Name, "Name of the target resource")

	cmd.Flags().StringVar(&restoreSessionOpt.repository, "repository", restoreSessionOpt.repository, "Name of the Repository")
	cmd.Flags().StringVar(&restoreSessionOpt.driver, "driver", restoreSessionOpt.driver, "Driver indicates the mechanism used to backup (i.e. VolumeSnapshotter, Restic)")
	cmd.Flags().StringVar(&restoreSessionOpt.task, "task", restoreSessionOpt.task, "Name of the Task")
	cmd.Flags().Int32Var(&restoreSessionOpt.replica, "replica", restoreSessionOpt.replica, "Replica specifies the number of replicas whose data should be backed up")

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

func createRestoreSession(name string, namespace string) (restoreSession *v1beta1.RestoreSession, err error) {

	restoreSession = &v1beta1.RestoreSession{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if v1beta1.Snapshotter(restoreSessionOpt.driver) == v1beta1.VolumeSnapshotter {
         restoreSession.Spec.Driver = v1beta1.Snapshotter(restoreSessionOpt.driver)
	} else {
		restoreSession.Spec = v1beta1.RestoreSessionSpec{
			Task:       v1beta1.TaskRef{Name: restoreSessionOpt.task},
			Rules:      append(make([]v1beta1.Rule, 0), restoreSessionOpt.rule),
			Repository: core.LocalObjectReference{Name: restoreSessionOpt.repository},
		}
	}

	err = setRestoreTarget(restoreSession)
	if err != nil {
		return nil, err
	}

	restoreSession, _, err = v1beta1_util.CreateOrPatchRestoreSession(stashClient.StashV1beta1(), restoreSession.ObjectMeta, func(in *v1beta1.RestoreSession) *v1beta1.RestoreSession {
		in.Spec = restoreSession.Spec
		return in
	})
	return restoreSession, err

}

func setRestoreVolumeMounts(target *v1beta1.RestoreTarget) error {
	// extract volume and mount information
	// then configure the volumeMounts of the target
	volMounts := make([]core.VolumeMount, 0)
	for _, m := range restoreSessionOpt.volumeMounts {
		vol := strings.Split(m, ":")
		if len(vol) == 3 {
			volMounts = append(volMounts, core.VolumeMount{Name: vol[0], MountPath: vol[1], SubPath: vol[2]})
		} else if len(vol) == 2 {
			volMounts = append(volMounts, core.VolumeMount{Name: vol[0], MountPath: vol[1]})
		} else {
			return fmt.Errorf("invalid volume-mounts. use either 'volName:mountPath' or 'volName:mountPath:subPath' format")
		}
	}
	target.VolumeMounts = volMounts
	return nil
}

func setRestoreTarget(restoreSession *v1beta1.RestoreSession) error {
	// if driver is VolumeSnapshotter then configure the Replica and VolumeClaimTemplates field
	// otherwise configure the TargetRef and replica field of the RestoreSession.
	if v1beta1.Snapshotter(restoreSessionOpt.driver) == v1beta1.VolumeSnapshotter {
		restoreSession.Spec.Target = &v1beta1.RestoreTarget{
			VolumeClaimTemplates: getRestoredPVCTemplates(),
		}
	} else {
		restoreSession.Spec.Target = &v1beta1.RestoreTarget{
			Ref: restoreSessionOpt.targetRef,
			VolumeClaimTemplates: getRestoredPVCTemplates(),
		}
		err := setRestoreVolumeMounts(restoreSession.Spec.Target)
		if err != nil {
			return err
		}
	}
	if restoreSessionOpt.replica > 0 {
		restoreSession.Spec.Target.Replicas = &restoreSessionOpt.replica
	}
	return nil
}

func getRestoredPVCTemplates() []core.PersistentVolumeClaim {
	return []core.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      restoreSessionOpt.volumeClaimTemplate.name,
				Namespace: namespace,
				CreationTimestamp: metav1.Time{
					Time: time.Now(),
				},
			},
			Spec: core.PersistentVolumeClaimSpec{
				AccessModes:      getPVAccessModes(restoreSessionOpt.volumeClaimTemplate.accessModes),
				StorageClassName: &restoreSessionOpt.volumeClaimTemplate.storageClass,
				Resources: core.ResourceRequirements{
					Requests: core.ResourceList{
						core.ResourceName(core.ResourceStorage): resource.MustParse(restoreSessionOpt.volumeClaimTemplate.size),
					},
				},
				DataSource: &core.TypedLocalObjectReference{
					Kind:     "VolumeSnapshot",
					Name:     restoreSessionOpt.volumeClaimTemplate.dataSource,
					APIGroup: types.StringP(vs.GroupName),
				},
			},
		},
	}
}

func getPVAccessModes(acModes []string) []core.PersistentVolumeAccessMode {
	accessModes := make([]core.PersistentVolumeAccessMode, 0)
	for _, am := range acModes {
		accessModes = append(accessModes, core.PersistentVolumeAccessMode(am))
	}
	return accessModes
}
