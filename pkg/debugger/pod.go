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

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	v1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	core_util "kmodules.xyz/client-go/core/v1"
)

func (opt *options) getWorkloadPods(targetRef v1beta1.TargetRef) (*core.PodList, error) {
	var matchLabels string
	switch targetRef.Kind {
	case apis.KindDeployment:
		deployment, err := opt.kubeClient.AppsV1().Deployments(opt.namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		matchLabels = labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels).String()
	case apis.KindStatefulSet:
		statefulset, err := opt.kubeClient.AppsV1().StatefulSets(opt.namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		matchLabels = labels.SelectorFromSet(statefulset.Spec.Selector.MatchLabels).String()
	case apis.KindDaemonSet:
		daemonset, err := opt.kubeClient.AppsV1().DaemonSets(opt.namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		matchLabels = labels.SelectorFromSet(daemonset.Spec.Selector.MatchLabels).String()
	}

	return opt.kubeClient.CoreV1().Pods(opt.namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: matchLabels,
	})
}

func (opt *options) getOwnedPods(job *v1.Job) ([]core.Pod, error) {
	podList, err := opt.kubeClient.CoreV1().Pods(opt.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var pods []core.Pod
	for i := range podList.Items {
		if owned, _ := core_util.IsOwnedBy(&podList.Items[i], job); owned {
			pods = append(pods, podList.Items[i])
		}
	}
	return pods, nil
}

func (opt *options) debugPod(pod *core.Pod) error {
	if err := opt.describeObject(pod.Name, apis.KindPod); err != nil {
		return err
	}
	return showLogs(pod, "--all-containers")
}
