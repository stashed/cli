package debugger

import (
	"context"

	"stash.appscode.dev/apimachinery/apis"

	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
)

func (opt *DebugOptions) getOwnedJobs(owner metav1.Object) ([]v1.Job, error) {
	jobList, err := opt.KubeClient.BatchV1().Jobs(opt.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var jobs []v1.Job
	for _, job := range jobList.Items {
		owned, _ := core_util.IsOwnedBy(&job, owner)
		if owned {
			jobs = append(jobs, job)
		}
	}
	return jobs, err
}

func (opt *DebugOptions) debugJob(session metav1.Object) error {
	jobs, err := opt.getOwnedJobs(session)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		pod, err := opt.getOwnedPod(&job)
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
