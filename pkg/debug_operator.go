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
	"stash.appscode.dev/cli/pkg/debugger"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/templates"
)

var debugOperatorExample = templates.Examples(`
		# Debug operator pod
		stash debug operator --namespace=<namespace>
        stash debug operator -n kube-system`)

func NewCmdDebugOperator() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "operator",
		Short:             `Debug Stash operator`,
		Long:              `Show debugging information for Stash operator`,
		Example:           debugOperatorExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dbgr := debugger.NewDebugger(kubeClient, stashClient, aggrClient, namespace)
			if err := dbgr.ShowVersionInformation(); err != nil {
				return err
			}
			return dbgr.DebugOperator()
		},
	}
	return cmd
}
