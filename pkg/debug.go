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
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	aggcs "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

func NewCmdDebug(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "debug",
		Short:             `Debug common Stash issues`,
		DisableAutoGenTag: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := clientGetter.ToRESTConfig()
			if err != nil {
				return errors.Wrap(err, "failed to read kubeconfig")
			}

			namespace, _, err = clientGetter.ToRawKubeConfigLoader().Namespace()
			if err != nil {
				return err
			}

			kubeClient, err = kubernetes.NewForConfig(cfg)
			if err != nil {
				return err
			}

			stashClient, err = cs.NewForConfig(cfg)
			if err != nil {
				return err
			}

			aggrClient, err = aggcs.NewForConfig(cfg)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.AddCommand(NewCmdDebugBackup())
	cmd.AddCommand(NewCmdDebugRestore())
	cmd.AddCommand(NewCmdDebugOperator())
	return cmd
}
