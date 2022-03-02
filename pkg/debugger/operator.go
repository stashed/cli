package debugger

import (
	"context"
	"fmt"
	"strings"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (opt *DebugOptions) GetOperatorPod() (*core.Pod, error) {
	apiSvc, err := opt.AggrClient.ApiregistrationV1beta1().APIServices().Get(context.TODO(), "v1alpha1.admission.stash.appscode.com", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	podList, err := opt.KubeClient.CoreV1().Pods(apiSvc.Spec.Service.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, pod := range podList.Items {
		if strings.Contains(pod.Name, "stash") {
			for _, c := range pod.Spec.Containers {
				if c.Name == "operator" {
					return &pod, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("operator pod not found")
}

func (opt *DebugOptions) DebugOperator() error {
	pod, err := opt.GetOperatorPod()
	if err != nil {
		return err
	}
	if err := showLogs(pod, "-c", "operator"); err != nil {
		return err
	}
	return nil
}
