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

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/spf13/cobra"
	"gomodules.xyz/go-sh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	debugBackupExample = templates.Examples(`
		# Debug a BackupConfigration
		stash debug backup --namespace=<namespace> --backupconfig=<backup-configuration-name>
        stash debug backup -n demo --backup-config=sample-mongodb`)
)

const (
	BackupExecutorSidecar   = "sidecar"
	BackupExecutorCSIDriver = "csi-driver"
	BackupExecutorJob       = "job"
)

type backupInvoker struct {
	name       string
	namespace  string
	kind       string
	phase      v1beta1.BackupInvokerPhase
	driver     v1beta1.Snapshotter
	targetRefs []v1beta1.TargetRef
}

func NewCmdDebugBackup() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "backup",
		Short:             `Debug backup`,
		Long:              `Debug backup by describing and showing logs of backup resources`,
		Example:           debugBackupExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			invoker := backupInvoker{}
			if backupConfig == "" && backupBatch == "" {
				return fmt.Errorf("neither BackupConfiguration nor BackupBatch name has been provided")
			}

			if err := showVersionInformation(); err != nil {
				return err
			}

			if err := invoker.getBackupInvokerInfo(); err != nil {
				return err
			}

			if err := invoker.debugBackup(); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func showVersionInformation() error {
	klog.Infoln("\n\n===============[ Client and Server version information ]===============")
	if err := sh.Command("kubectl", "version", "--short").Run(); err != nil {
		return err
	}

	pod, err := GetOperatorPod()
	if err != nil {
		return err
	}
	klog.Infoln("\n\n===============[ Operator's binary version number ]===============")

	if err := sh.Command("kubectl", "exec", "-it", "-n", pod.Namespace, pod.Name, "-c", "operator", "--", "/stash-enterprise", "version").Run(); err != nil {
		return err
	}
	return nil
}

func (inv *backupInvoker) getBackupInvokerInfo() error {
	inv.namespace = namespace
	if backupConfig != "" {
		bc, err := stashClient.StashV1beta1().BackupConfigurations(namespace).Get(context.TODO(), backupConfig, metav1.GetOptions{})
		if err != nil {
			return err
		}

		inv.name = bc.Name
		inv.kind = v1beta1.ResourceKindBackupConfiguration
		inv.driver = bc.Spec.Driver
		inv.phase = bc.Status.Phase
		inv.targetRefs = append(inv.targetRefs, bc.Spec.Target.Ref)

	} else {
		bb, err := stashClient.StashV1beta1().BackupBatches(namespace).Get(context.TODO(), backupBatch, metav1.GetOptions{})
		if err != nil {
			return err
		}
		inv.name = bb.Name
		inv.kind = v1beta1.ResourceKindBackupBatch
		inv.driver = bb.Spec.Driver
		inv.phase = bb.Status.Phase
		for _, member := range bb.Spec.Members {
			inv.targetRefs = append(inv.targetRefs, member.Target.Ref)
		}
	}

	return nil
}

func (inv *backupInvoker) debugBackup() error {
	if inv.phase == v1beta1.BackupInvokerReady {
		backupsessionList, err := stashClient.StashV1beta1().BackupSessions(inv.namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, backupsession := range backupsessionList.Items {
			klog.Infoln(backupsession.OwnerReferences[0].Name, backupsession.OwnerReferences[0].Kind, inv.name, inv.kind)
			if backupsession.Status.Phase == v1beta1.BackupSessionFailed &&
				backupsession.OwnerReferences[0].Name == inv.name &&
				backupsession.OwnerReferences[0].Kind == inv.kind {
				klog.Infof("\n\n===============[ Describing Backupsession: %s ]===============", backupsession.Name)
				if err := describeBackupsession(&backupsession); err != nil {
					return err
				}
			}
			for _, targetRef := range inv.targetRefs {
				executor := inv.getBackupExecutor(targetRef)
				if executor == apis.ModelSidecar {

				} else {

				}
			}
		}

	} else {
		if err := inv.describeBackupInvoker(); err != nil {
			return err
		}
	}
	return nil
}

func describeBackupsession(bs *v1beta1.BackupSession) error {
	if err := sh.Command("kubectl", "describe", "Backupsession", "-n", bs.Namespace, bs.Name).Run(); err != nil {
		return err
	}
	return nil
}

func (inv *backupInvoker) getBackupExecutor(tref v1beta1.TargetRef) string {
	if inv.driver == v1beta1.ResticSnapshotter &&
		util.BackupModel(tref.Kind) == apis.ModelSidecar {
		return BackupExecutorSidecar
	}
	if inv.driver == v1beta1.VolumeSnapshotter {
		return BackupExecutorCSIDriver
	}
	return BackupExecutorJob
}

func (inv *backupInvoker) describeBackupInvoker() error {

	klog.Infof("\n\n===============[ Describe BackupInvoker ]===============")
	if err := sh.Command("kubectl", "describe", inv.kind, "-n", inv.namespace, inv.name).Run(); err != nil {
		return err
	}
	return nil
}
