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
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	v1beta1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"strings"
	"time"
)

var (
	createRestoreSessionLong = templates.LongDesc(`
		Create a new RestoreSession`)

	createRestoreSessionExample = templates.Examples(`
		# Create a RestoreSession
		# stash create restore --namespace=demo <restore session name> [Flag]
        # For Restic driver
         stash create restore ss-restore --namespace=demo --repository=gcs-repo --target-apiversion=apps/v1 --target-kind=StatefulSet --target-name=stash-recovered-adv --paths=/source/data --volume-mounts=source-data:/source/data
        # For VolumeSnapshotter driver
         stash create restore restore-pvc --namespace=demo --driver=VolumeSnapshotter --replica=3 --vctpl-name=restore-data-restore-demo-${POD_ORDINAL} --vctpl-accessmode=ReadWriteOnce --vctpl-storageclass=standard --vctpl-request-size=1Gi --vctpl-datasource-name=source-data-stash-demo-0-1567146010`)
)

func NewCmdCreateRestoreSession() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "restore",
		Short:             `Create a RestoreSession`,
		Long:              createRestoreSessionLong,
		Example:           createRestoreSessionExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("RestoreSession name has not provided")
			}

			restoresessionName := args[0]

			restoreSession, err := createRestoreSession(restoresessionName)
			if err != nil {
				return err
			}
			log.Infof("RestoreSession %s/%s has been created successfully.", restoreSession.Namespace, restoreSession.Name)
			return err

		},
	}

	cmd.Flags().StringVar(&opt.TargetRef.APIVersion, "target-apiversion", opt.TargetRef.APIVersion, "API-Version of the target resource")
	cmd.Flags().StringVar(&opt.TargetRef.Kind, "target-kind", opt.TargetRef.Kind, "Kind of the target resource")
	cmd.Flags().StringVar(&opt.TargetRef.Name, "target-name", opt.TargetRef.Name, "Name of the target resource")

	cmd.Flags().StringVar(&opt.RepositoryName, "repository", opt.RepositoryName, "Name of the Repository")
	cmd.Flags().StringVar(&opt.Driver, "driver", opt.Driver, "Driver indicates the mechanism used to backup (i.e. VolumeSnapshotter, Restic)")
	cmd.Flags().StringVar(&opt.TaskName, "task", opt.TaskName, "Name of the Task")
	cmd.Flags().Int32Var(&opt.Replica, "replica", opt.Replica, "Replica specifies the number of replicas whose data should be backed up")

	cmd.Flags().StringVar(&opt.Paths, "paths", opt.Paths, "List of paths to backup")
	cmd.Flags().StringVar(&opt.VolMounts, "volume-mounts", opt.VolMounts, "List of volumes and their mountPaths")

	cmd.Flags().StringVar(&opt.Rule.SourceHost, "source-host", opt.Rule.SourceHost, "Name of the Source host")
	cmd.Flags().StringVar(&opt.Rule.Snapshot, "snapshot", opt.Rule.Snapshot, "Name of the Snapshot(single)")

	cmd.Flags().StringVar(&opt.VolumeClaimTemp.Name, "vctpl-name", opt.VolumeClaimTemp.Name, "Name of the VolumeClaimTemplate")
	cmd.Flags().StringVar(&opt.VolumeClaimTemp.AccessMode, "vctpl-accessmode", opt.VolumeClaimTemp.AccessMode, "Access mode of the VolumeClaimTemplates")
	cmd.Flags().StringVar(&opt.VolumeClaimTemp.StorageClassName, "vctpl-storageclass", opt.VolumeClaimTemp.StorageClassName, "Name of the Storage secret for VolumeClaimTemplate")
	cmd.Flags().StringVar(&opt.VolumeClaimTemp.Size, "vctpl-request-size", opt.VolumeClaimTemp.Size, "Total requested size of the VolumeClaimTemplate")
	cmd.Flags().StringVar(&opt.VolumeClaimTemp.DataSourceName, "vctpl-datasource-name", opt.VolumeClaimTemp.DataSourceName, "DataSource of the VolumeClaimTemplate")

	return cmd
}

func createRestoreSession(name string) (restoreSesstion *v1beta1.RestoreSession, err error) {
	// Configure VolumeClaimTemplate
	if v1beta1.Snapshotter(opt.Driver) == v1beta1.VolumeSnapshotter {
		err = configureVolumeClaimTemplate()
		if err != nil {
			return restoreSesstion, err
		}
	} else {
		// Configure VolumeMounts and Paths
		err = configureVolumeMountsAndPathsOrSnapshots()
		if err != nil {
			return restoreSesstion, err
		}
	}
	restoreSesstion = restoreSessionObj(name)
	restoreSesstion, _, err = v1beta1_util.CreateOrPatchRestoreSession(stashClient.StashV1beta1(),
		metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		func(obj *v1beta1.RestoreSession) *v1beta1.RestoreSession {
			obj.Spec = restoreSesstion.Spec
			return obj
		})
	return restoreSesstion, err

}

func configureVolumeClaimTemplate() error {

	accessModes, err := getPVAccessModes(opt.VolumeClaimTemp.AccessMode)
	if err != nil {
		return err
	}
	volumeClaimTemplate = core.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: opt.VolumeClaimTemp.Name,
			Namespace: namespace,
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
		},
		Spec: core.PersistentVolumeClaimSpec{
			AccessModes:      accessModes,
			StorageClassName: &opt.VolumeClaimTemp.StorageClassName,
			Resources: core.ResourceRequirements{
				Requests: core.ResourceList{
					core.ResourceName(core.ResourceStorage): resource.MustParse(opt.VolumeClaimTemp.Size),
				},
			},
			DataSource: &core.TypedLocalObjectReference{
				Kind: "VolumeSnapshot",
				Name: opt.VolumeClaimTemp.DataSourceName,
				APIGroup: types.StringP(vs.GroupName),
			},
		},
	}
	return nil
}

func restoreSessionObj(name string) *v1beta1.RestoreSession {
	restoreSession := &v1beta1.RestoreSession{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1beta1.ResourceKindRestoreSession,
			APIVersion: v1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if v1beta1.Snapshotter(opt.Driver) == v1beta1.VolumeSnapshotter {
		restoreSession.Spec = v1beta1.RestoreSessionSpec{
			Driver: v1beta1.Snapshotter(opt.Driver),
			Target: &v1beta1.RestoreTarget{
				Replicas:  &opt.Replica,
				VolumeClaimTemplates: []core.PersistentVolumeClaim{
					{
						Spec: volumeClaimTemplate.Spec,
						ObjectMeta: volumeClaimTemplate.ObjectMeta,
					},
				},
			},
		}
	} else {
		restoreSession.Spec.Repository = core.LocalObjectReference{Name: opt.RepositoryName}
		restoreSession.Spec.Target = &v1beta1.RestoreTarget{
			Ref:          opt.TargetRef,
			VolumeMounts: volumeMounts,
		}
		restoreSession.Spec.Task = v1beta1.TaskRef{Name: opt.TaskName}
		rule := make([]v1beta1.Rule, 0)
		if opt.Rule.SourceHost != "" {
			if opt.Rule.Snapshot != "" {
				rule = append(rule, v1beta1.Rule{SourceHost: opt.Rule.SourceHost, Snapshots: snapshots})
			} else {
				rule = append(rule, v1beta1.Rule{SourceHost: opt.Rule.SourceHost, Paths: paths})
			}
			restoreSession.Spec.Rules = rule
		} else {
			if opt.Rule.Snapshot != "" {
				rule = append(rule, v1beta1.Rule{Snapshots: snapshots})
			} else {
				rule = append(rule, v1beta1.Rule{Paths: paths})
			}
			restoreSession.Spec.Rules = rule
		}
	}
	return restoreSession
}

func getPVAccessModes(str string) (accessModes []core.PersistentVolumeAccessMode, err error) {
	accessModes = make([]core.PersistentVolumeAccessMode, 0)
	strArray := strings.Split(str, ",")
	for _, acMode := range strArray {
		switch core.PersistentVolumeAccessMode(acMode) {
		case core.ReadOnlyMany:
			accessModes = append(accessModes, core.PersistentVolumeAccessMode(acMode))
		case core.ReadWriteOnce:
			accessModes = append(accessModes, core.PersistentVolumeAccessMode(acMode))
		case core.ReadWriteMany:
			accessModes = append(accessModes, core.PersistentVolumeAccessMode(acMode))
		default:
			return accessModes, fmt.Errorf("VolumeClaimTemplate Access Mode are not defined properly")
		}
	}
	return accessModes, nil
}
