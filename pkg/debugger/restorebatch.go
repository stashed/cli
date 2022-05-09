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
	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"

	"k8s.io/klog/v2"
)

func (opt *options) DebugRestoreBatch(restoreBatch *v1beta1.RestoreBatch) error {
	if err := opt.describeObject(restoreBatch.Name, v1beta1.ResourceKindRestoreBatch); err != nil {
		return err
	}
	if restoreBatch.Status.Phase == v1beta1.RestorePending {
		return nil
	}
	for _, member := range restoreBatch.Spec.Members {
		klog.Infof("\n\n\n\n\n\n==================[ Debugging backup for target %s %s/%s ]==================", member.Target.Ref.Kind, opt.namespace, member.Target.Ref.Name)
		if util.RestoreModel(member.Target.Ref.Kind, member.Task.Name) == apis.ModelSidecar {
			if err := opt.debugSidecar(member.Target.Ref, apis.StashInitContainer); err != nil {
				return err
			}
		} else {
			if err := opt.debugJobs(restoreBatch); err != nil {
				return err
			}
		}
	}
	return nil
}
