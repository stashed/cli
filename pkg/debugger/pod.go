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

func (opt *DebugOptions) getWorkloadPods(targetRef v1beta1.TargetRef) (*core.PodList, error) {
	var matchLabels string
	switch targetRef.Kind {
	case apis.KindDeployment:
		deployment, err := opt.KubeClient.AppsV1().Deployments(opt.Namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		matchLabels = labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels).String()
	case apis.KindStatefulSet:
		statefulset, err := opt.KubeClient.AppsV1().StatefulSets(opt.Namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		matchLabels = labels.SelectorFromSet(statefulset.Spec.Selector.MatchLabels).String()
	case apis.KindDaemonSet:
		daemonset, err := opt.KubeClient.AppsV1().DaemonSets(opt.Namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		matchLabels = labels.SelectorFromSet(daemonset.Spec.Selector.MatchLabels).String()
	}

	podList, err := opt.KubeClient.CoreV1().Pods(opt.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: matchLabels,
	})
	if err != nil {
		return nil, err
	}
	return podList, nil
}

func (opt *DebugOptions) getOwnedPod(job *v1.Job) (*core.Pod, error) {
	podList, err := opt.KubeClient.CoreV1().Pods(opt.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, pod := range podList.Items {
		owned, _ := core_util.IsOwnedBy(&pod, job)
		if owned {
			return &pod, nil
		}
	}
	return nil, nil
}
