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

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/spf13/cobra"
	"gomodules.xyz/go-sh"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	debugBackupExample = templates.Examples(`
		# Debug a BackupConfigration
		stash debug backup --namespace=<namespace> --backupconfig=<backupconfiguration-name>
        stash debug backup --namespace=demo --backupconfig=sample-mongodb-backup`)
)

func NewCmdDebugBackup() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "backup",
		Short:             `Debug backup`,
		Long:              `Debug common Stash backup issues`,
		Example:           debugBackupExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if backupConfig == "" && backupBatch == "" {
				return fmt.Errorf("neither BackupConfiguration nor BackupBatch name has been provided")
			}

			if err := showVersionInformation(); err != nil {
				return err
			}

			if backupConfig != "" {
				bc, err := stashClient.StashV1beta1().BackupConfigurations(namespace).Get(context.TODO(), backupConfig, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if err := debugBackupConfig(bc); err != nil {
					return err
				}
			} else {
				bb, err := stashClient.StashV1beta1().BackupBatches(namespace).Get(context.TODO(), backupBatch, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if err := debugBackupBatch(bb); err != nil {
					return err
				}

			}
			return nil
		},
	}
	cmd.Flags().StringVar(&backupConfig, "backupconfig", backupConfig, "Name of the BackupConfiguration to debug")
	cmd.Flags().StringVar(&backupBatch, "backupbatch", backupBatch, "Name of the BackupBatch to debug")
	return cmd
}

func showVersionInformation() error {
	klog.Infoln("\n\n\n===============[ Kubernetes Version ]===============")
	if err := sh.Command("kubectl", "version", "--short").Run(); err != nil {
		return err
	}

	pod, err := GetOperatorPod()
	if err != nil {
		return err
	}
	var stashBinary string
	if strings.Contains(pod.Name, "stash-enterprise") {
		stashBinary = "/stash-enterprise"
	} else {
		stashBinary = "/stash"
	}
	klog.Infoln("\n\n\n===============[ Stash Version ]===============")
	if err := sh.Command("kubectl", "exec", "-it", "-n", pod.Namespace, pod.Name, "-c", "operator", "--", stashBinary, "version").Run(); err != nil {
		return err
	}
	return nil
}

func debugBackupConfig(backupConfig *v1beta1.BackupConfiguration) error {
	if backupConfig.Status.Phase == v1beta1.BackupInvokerReady {
		backupSessions, err := getOwnedBackupSessions(backupConfig)
		if err != nil {
			return err
		}
		for _, backupSession := range backupSessions {
			if err := debugBackupSession(&backupSession, []v1beta1.BackupConfigurationTemplateSpec{backupConfig.Spec.BackupConfigurationTemplateSpec}); err != nil {
				return err
			}
		}
	} else {
		if err := describeObject(backupConfig, v1beta1.ResourceKindBackupConfiguration); err != nil {
			return err
		}
	}
	return nil
}

func debugBackupBatch(backupBatch *v1beta1.BackupBatch) error {
	if backupBatch.Status.Phase == v1beta1.BackupInvokerReady {
		backupSessions, err := getOwnedBackupSessions(backupBatch)
		if err != nil {
			return err
		}
		for _, backupSession := range backupSessions {
			if err := debugBackupSession(&backupSession, backupBatch.Spec.Members); err != nil {
				return err
			}
		}
	} else {
		if err := describeObject(backupBatch, v1beta1.ResourceKindBackupBatch); err != nil {
			return err
		}
	}
	return nil
}

func debugBackupSession(backupSession *v1beta1.BackupSession, members []v1beta1.BackupConfigurationTemplateSpec) error {
	if backupSession.Status.Phase == v1beta1.BackupSessionFailed {
		if err := describeObject(backupSession, v1beta1.ResourceKindBackupSession); err != nil {
			return err
		}
		for _, member := range members {
			klog.Infof("\n\n\n\n\n\n===============[ Debugging backup for %s ]===============", member.Target.Ref.Name)
			backupModel := util.BackupModel(member.Target.Ref.Kind)
			if backupModel == apis.ModelSidecar {
				if err := debugSidecar(member.Target.Ref, apis.StashContainer); err != nil {
					return err
				}
			} else if backupModel == apis.ModelCronJob {
				if err := debugJob(backupSession); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func debugSidecar(targetRef v1beta1.TargetRef, container string) error {
	workloadPods, err := getWorkloadPods(targetRef)
	if err != nil {
		return err
	}
	for _, workloadPods := range workloadPods.Items {
		if err := showLogs(&workloadPods, "-c", container); err != nil {
			return err
		}
	}
	return nil
}

func debugJob(session metav1.Object) error {
	jobs, err := getOwnedJobs(session)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		pod, err := getOwnedPod(&job)
		if err != nil {
			return err
		}
		if pod != nil {
			if err := describeObject(pod, apis.KindPod); err != nil {
				return err
			}
			if err := showLogs(pod, "--all-containers"); err != nil {
				return err
			}
		}
	}
	return nil
}

func showLogs(pod *core.Pod, args ...string) error {
	klog.Infof("\n\n\n\n\n\n===============[ Logs from pod: %s ]===============", pod.Name)
	cmdArgs := []string{"logs", "-n", pod.Namespace, pod.Name}
	cmdArgs = append(cmdArgs, args...)
	if err := sh.Command("kubectl", cmdArgs).Run(); err != nil {
		return err
	}
	return nil
}

func describeObject(object metav1.Object, resourceKind string) error {
	klog.Infof("\n\n\n\n\n\n===============[ Describing %s: %s ]===============", resourceKind, object.GetName())
	if err := sh.Command("kubectl", "describe", resourceKind, "-n", object.GetNamespace(), object.GetName()).Run(); err != nil {
		return err
	}
	return nil
}
