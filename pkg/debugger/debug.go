package debugger

import (
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"

	"k8s.io/client-go/kubernetes"
	"k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

type DebugOptions struct {
	KubeClient  *kubernetes.Clientset
	StashClient *cs.Clientset
	AggrClient  *clientset.Clientset
	Namespace   string
}
