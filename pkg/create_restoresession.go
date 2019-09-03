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
	"time"
)

var (
	createRestoreSessionExample = templates.Examples(`
		# Create a RestoreSession
		# stash create restore --namespace=demo <restore session name> [Flag]
        # For Restic driver
         stash create restoresession ss-restore --namespace=demo --repository=gcs-repo --target-apiversion=apps/v1 --target-kind=StatefulSet --target-name=stash-recovered-adv --paths=/source/data --volume-mounts=source-data:/source/data
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

	rule                rule
	volumeClaimTemplate volumeclaimTemplate
}

type volumeclaimTemplate struct {
	name         string
	accessModes  []string
	storageClass string
	size         string
	dataSource   string
}

type rule struct {
	sourceHost string
	snapshots  []string
	paths      []string
}

func NewCmdCreateRestoreSession() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "restoresession",
		Short:             `Create a new RestoreSession`,
		Long:              `Create a new RestoreSession using target resource or PVC Template`,
		Example:           createRestoreSessionExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("RestoreSession name is not provided")
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

	cmd.Flags().StringVar(&restoreSessionOpt.targetRef.APIVersion, "target-apiversion", restoreSessionOpt.targetRef.APIVersion, "API-Version of the target resource")
	cmd.Flags().StringVar(&restoreSessionOpt.targetRef.Kind, "target-kind", restoreSessionOpt.targetRef.Kind, "Kind of the target resource")
	cmd.Flags().StringVar(&restoreSessionOpt.targetRef.Name, "target-name", restoreSessionOpt.targetRef.Name, "Name of the target resource")

	cmd.Flags().StringVar(&restoreSessionOpt.repository, "repository", restoreSessionOpt.repository, "Name of the Repository")
	cmd.Flags().StringVar(&restoreSessionOpt.driver, "driver", restoreSessionOpt.driver, "Driver indicates the mechanism used to backup (i.e. VolumeSnapshotter, Restic)")
	cmd.Flags().StringVar(&restoreSessionOpt.task, "task", restoreSessionOpt.task, "Name of the Task")
	cmd.Flags().Int32Var(&restoreSessionOpt.replica, "replica", restoreSessionOpt.replica, "Replica specifies the number of replicas whose data should be backed up")

	cmd.Flags().StringSliceVar(&restoreSessionOpt.volumeMounts, "volume-mounts", restoreSessionOpt.volumeMounts, "List of volumes and their mountPaths")
	cmd.Flags().StringSliceVar(&restoreSessionOpt.rule.paths, "paths", restoreSessionOpt.rule.paths, "List of paths to backup")
	cmd.Flags().StringSliceVar(&restoreSessionOpt.rule.snapshots, "snapshots", restoreSessionOpt.rule.snapshots, "Name of the Snapshot(single)")
	cmd.Flags().StringVar(&restoreSessionOpt.rule.sourceHost, "host", restoreSessionOpt.rule.sourceHost, "Name of the Source host")
	cmd.Flags().StringVar(&restoreSessionOpt.volumeClaimTemplate.name, "claim.name", restoreSessionOpt.volumeClaimTemplate.name, "Name of the VolumeClaimTemplate")
	cmd.Flags().StringSliceVar(&restoreSessionOpt.volumeClaimTemplate.accessModes, "claim.access-modes", restoreSessionOpt.volumeClaimTemplate.accessModes, "Access mode of the VolumeClaimTemplates")
	cmd.Flags().StringVar(&restoreSessionOpt.volumeClaimTemplate.storageClass, "claim.storageclass", restoreSessionOpt.volumeClaimTemplate.storageClass, "Name of the Storage secret for VolumeClaimTemplate")
	cmd.Flags().StringVar(&restoreSessionOpt.volumeClaimTemplate.size, "claim.size", restoreSessionOpt.volumeClaimTemplate.size, "Total requested size of the VolumeClaimTemplate")
	cmd.Flags().StringVar(&restoreSessionOpt.volumeClaimTemplate.dataSource, "claim.datasource", restoreSessionOpt.volumeClaimTemplate.dataSource, "DataSource of the VolumeClaimTemplate")

	return cmd
}

func createRestoreSession(name string) (restoreSession *v1beta1.RestoreSession, err error) {

	restoreSession = &v1beta1.RestoreSession{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1beta1.ResourceKindRestoreSession,
			APIVersion: v1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	err = setRestoreTarget(restoreSession)
	if err != nil {
		return restoreSession, err
	}

	restoreSession, _, err = v1beta1_util.CreateOrPatchRestoreSession(stashClient.StashV1beta1(), restoreSession.ObjectMeta, func(obj *v1beta1.RestoreSession) *v1beta1.RestoreSession {
		obj.Spec = restoreSession.Spec
		return obj
	})
	return restoreSession, err

}

func setRestoreTarget(restoreSession *v1beta1.RestoreSession) error {
	// if driver is VolumeSnapshotter then configure the Driver, Replica and VolumeClaimTemplates field
	// otherwise configure the TargetRef, Rules field of the RestoreSession.
	if v1beta1.Snapshotter(restoreSessionOpt.driver) == v1beta1.VolumeSnapshotter {
		restoreSession.Spec = v1beta1.RestoreSessionSpec {
			Driver: v1beta1.Snapshotter(restoreSessionOpt.driver),
			Target: &v1beta1.RestoreTarget{
				Replicas: &restoreSessionOpt.replica,
				VolumeClaimTemplates: []core.PersistentVolumeClaim {
					{
						ObjectMeta: metav1.ObjectMeta {
							Name:      restoreSessionOpt.volumeClaimTemplate.name,
							Namespace: namespace,
							CreationTimestamp: metav1.Time {
								Time: time.Now(),
							},
						},
						Spec: core.PersistentVolumeClaimSpec {
							AccessModes:      setPVAccessModes(restoreSessionOpt.volumeClaimTemplate.accessModes),
							StorageClassName: &restoreSessionOpt.volumeClaimTemplate.storageClass,
							Resources: core.ResourceRequirements {
								Requests: core.ResourceList {
									core.ResourceName(core.ResourceStorage): resource.MustParse(restoreSessionOpt.volumeClaimTemplate.size),
								},
							},
							DataSource: &core.TypedLocalObjectReference {
								Kind:     "VolumeSnapshot",
								Name:     restoreSessionOpt.volumeClaimTemplate.dataSource,
								APIGroup: types.StringP(vs.GroupName),
							},
						},
					},
				},
			},
		}
	} else {
		restoreSession.Spec.Repository = core.LocalObjectReference{Name: restoreSessionOpt.repository}
		restoreSession.Spec.Target = &v1beta1.RestoreTarget{
			Ref: restoreSessionOpt.targetRef,
		}
		err := setVolumeMounts(restoreSession.Spec.Target)
		if err != nil {
			return err
		}
		restoreSession.Spec.Task = v1beta1.TaskRef{Name: restoreSessionOpt.task}
		rule := make([]v1beta1.Rule, 0)
		if restoreSessionOpt.rule.sourceHost != "" {
			if len(restoreSessionOpt.rule.snapshots) > 0 {
				rule = append(rule, v1beta1.Rule{SourceHost: restoreSessionOpt.rule.sourceHost, Snapshots: restoreSessionOpt.rule.snapshots})
			} else {
				rule = append(rule, v1beta1.Rule{SourceHost: restoreSessionOpt.rule.sourceHost, Paths: restoreSessionOpt.rule.paths})
			}
		} else {
			if len(restoreSessionOpt.rule.snapshots) > 0 {
				rule = append(rule, v1beta1.Rule{Snapshots: restoreSessionOpt.rule.snapshots})
			} else {
				rule = append(rule, v1beta1.Rule{Paths: restoreSessionOpt.rule.paths})
			}
		}
		restoreSession.Spec.Rules = rule
	}
	return nil
}

func setPVAccessModes(acModes []string) []core.PersistentVolumeAccessMode {
	accessModes := make([]core.PersistentVolumeAccessMode, 0)
	for _, am := range acModes {
		accessModes = append(accessModes, core.PersistentVolumeAccessMode(am))
	}
	return accessModes
}
