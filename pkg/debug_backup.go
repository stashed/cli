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
	v1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/util/templates"
	core_util "kmodules.xyz/client-go/core/v1"
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
		Long:              `Debug backup by describing and showing logs of backup resources`,
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
	klog.Infoln("\n\n\n===============[ Client and Server version information ]===============")
	if err := sh.Command("kubectl", "version", "--short").Run(); err != nil {
		return err
	}

	pod, err := GetOperatorPod()
	if err != nil {
		return err
	}
	klog.Infoln("\n\n\n===============[ Operator's binary version information ]===============")

	if err := sh.Command("kubectl", "exec", "-it", "-n", pod.Namespace, pod.Name, "-c", "operator", "--", "/stash-enterprise", "version").Run(); err != nil {
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
		jobList, podList, err := getAllJobsAndPods()
		if err != nil {
			return err
		}
		for _, backupSession := range backupSessions {
			if backupSession.Status.Phase == v1beta1.BackupSessionFailed {
				if err := describeObject(&backupSession, v1beta1.ResourceKindBackupSession); err != nil {
					return err
				}
				backupModel := util.BackupModel(backupConfig.Spec.Target.Ref.Kind)
				if backupModel == apis.ModelSidecar {
					if err := debugWorkloadPods(backupConfig.Spec.Target.Ref, apis.StashContainer); err != nil {
						return err
					}

				} else if backupModel == apis.ModelCronJob {
					if err := debugBackupJob(&backupSession, jobList, podList); err != nil {
						return err
					}
				}
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
	klog.Infoln(backupBatch.Status.Phase)
	if backupBatch.Status.Phase == v1beta1.BackupInvokerReady {
		backupSessions, err := getOwnedBackupSessions(backupBatch)
		if err != nil {
			return err
		}
		jobList, podList, err := getAllJobsAndPods()
		if err != nil {
			return err
		}
		for _, backupSession := range backupSessions {
			if backupSession.Status.Phase == v1beta1.BackupSessionFailed {
				if err := describeObject(&backupSession, v1beta1.ResourceKindBackupSession); err != nil {
					return err
				}
				for _, member := range backupBatch.Spec.Members {
					backupModel := util.BackupModel(member.Target.Ref.Kind)
					if backupModel == apis.ModelSidecar {
						if err := debugWorkloadPods(member.Target.Ref, apis.StashContainer); err != nil {
							return err
						}
					} else if backupModel == apis.ModelCronJob {
						if err := debugBackupJob(&backupSession, jobList, podList); err != nil {
							return err
						}
					}
				}
			}
		}
	} else {
		if err := describeObject(backupBatch, v1beta1.ResourceKindBackupBatch); err != nil {
			return err
		}
	}
	return nil
}

func getOwnedBackupSessions(backupInvoker metav1.Object) ([]v1beta1.BackupSession, error) {
	backupSessionList, err := stashClient.StashV1beta1().BackupSessions(backupInvoker.GetNamespace()).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var ownedBackupsessions []v1beta1.BackupSession
	for _, backupSession := range backupSessionList.Items {
		owned, _ := core_util.IsOwnedBy(&backupSession, backupInvoker)
		if owned {
			ownedBackupsessions = append(ownedBackupsessions, backupSession)
		}
	}
	return ownedBackupsessions, nil
}

func getAllJobsAndPods() (*v1.JobList, *core.PodList, error) {
	jobList, err := kubeClient.BatchV1().Jobs(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	podList, err := kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	return jobList, podList, err
}

func describeObject(object metav1.Object, resourceKind string) error {
	klog.Infof("\n\n\n\n\n\n===============[ Describing %s: %s ]===============", resourceKind, object.GetName())
	if err := sh.Command("kubectl", "describe", resourceKind, "-n", object.GetNamespace(), object.GetName()).Run(); err != nil {
		return err
	}
	return nil
}

func debugWorkloadPods(targetRef v1beta1.TargetRef, container string) error {
	workloadPods, err := getWorkloadPods(targetRef)
	if err != nil {
		return err
	}
	for _, workloadPods := range workloadPods.Items {
		if err := showContainerLogs(&workloadPods, container); err != nil {
			return err
		}
	}
	return nil
}

func getWorkloadPods(targetRef v1beta1.TargetRef) (*core.PodList, error) {
	var podList *core.PodList
	if targetRef.Kind == apis.KindDeployment {
		deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		podList, err = kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels).String(),
		})
		if err != nil {
			return nil, err
		}
	} else if targetRef.Kind == apis.KindStatefulSet {
		statefulset, err := kubeClient.AppsV1().StatefulSets(namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		podList, err = kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(statefulset.Spec.Selector.MatchLabels).String(),
		})
		if err != nil {
			return nil, err
		}
	} else if targetRef.Kind == apis.KindDaemonSet {
		daemonset, err := kubeClient.AppsV1().DaemonSets(namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		podList, err = kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(daemonset.Spec.Selector.MatchLabels).String(),
		})
		if err != nil {
			return nil, err
		}
	}
	return podList, nil
}

func debugBackupJob(backupSession *v1beta1.BackupSession, jobList *v1.JobList, podList *core.PodList) error {
	backupJobs := getOwnedJobs(jobList, backupSession)
	for _, backupJob := range backupJobs {
		backupPod := getOwnedPod(podList, &backupJob)
		if backupPod != nil {
			if err := showAllContainersLogs(backupPod); err != nil {
				return err
			}
		}
	}
	return nil
}

func getOwnedJobs(jobList *v1.JobList, owner metav1.Object) []v1.Job {
	var backupJobs []v1.Job
	for _, job := range jobList.Items {
		owned, _ := core_util.IsOwnedBy(&job, owner)
		if owned {
			backupJobs = append(backupJobs, job)
		}
	}
	return backupJobs
}

func getOwnedPod(podList *core.PodList, job *v1.Job) *core.Pod {
	for _, pod := range podList.Items {
		owned, _ := core_util.IsOwnedBy(&pod, job)
		if owned {
			return &pod
		}
	}
	return nil
}

func showAllContainersLogs(pod *core.Pod) error {
	klog.Infof("\n\n\n\n\n\n===============[ Logs from pod: %s ]===============", pod.Name)
	if err := sh.Command("kubectl", "logs", "-n", pod.Namespace, "--all-containers", pod.Name).Run(); err != nil {
		return err
	}
	return nil
}

func showContainerLogs(pod *core.Pod, container string) error {
	klog.Infof("\n\n\n\n===============[ logs from pod: %s, container: %s ]===============", pod.Name, container)
	if err := sh.Command("kubectl", "logs", "-n", pod.Namespace, "-c", container, pod.Name).Run(); err != nil {
		return err
	}
	return nil
}
