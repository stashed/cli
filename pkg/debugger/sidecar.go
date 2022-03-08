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
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
)

func (opt *options) debugSidecar(targetRef v1beta1.TargetRef, container string) error {
	workloadPods, err := opt.getWorkloadPods(targetRef)
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
