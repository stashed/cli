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
	"strings"

	"gomodules.xyz/go-sh"
	"k8s.io/klog/v2"
)

func (opt *options) ShowVersionInformation() error {
	if err := showKubernetesVersion(); err != nil {
		return err
	}
	return opt.showStashVersion()
}

func showKubernetesVersion() error {
	klog.Infoln("\n\n\n==================[ Kubernetes Version ]==================")
	return sh.Command("kubectl", "version", "--short").Run()
}

func (opt *options) showStashVersion() error {
	pod, err := opt.getOperatorPod()
	if err != nil {
		return err
	}
	var stashBinary string
	if strings.Contains(pod.Name, "stash-enterprise") {
		stashBinary = "/stash-enterprise"
	} else {
		stashBinary = "/stash"
	}
	klog.Infoln("\n\n\n==================[ Stash Version ]==================")
	return sh.Command("kubectl", "exec", "-it", "-n", pod.Namespace, pod.Name, "-c", "operator", "--", stashBinary, "version").Run()
}
