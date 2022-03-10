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

package debugger

import (
	"context"

	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
)

func (opt *options) getOwnedJobs(owner metav1.Object) ([]v1.Job, error) {
	jobList, err := opt.kubeClient.BatchV1().Jobs(opt.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var jobs []v1.Job
	for i := range jobList.Items {
		if owned, _ := core_util.IsOwnedBy(&jobList.Items[i], owner); owned {
			jobs = append(jobs, jobList.Items[i])
		}
	}
	return jobs, nil
}

func (opt *options) debugJobs(owner metav1.Object) error {
	jobs, err := opt.getOwnedJobs(owner)
	if err != nil {
		return err
	}
	for i := range jobs {
		err := opt.debugJob(&jobs[i])
		if err != nil {
			return err
		}

	}
	return nil
}

func (opt *options) debugJob(job *v1.Job) error {
	pods, err := opt.getOwnedPods(job)
	if err != nil {
		return err
	}
	for i := range pods {
		if err := opt.debugPod(&pods[i]); err != nil {
			return err
		}
	}
	return nil
}
