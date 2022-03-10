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

	vs_cs "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

func NewCmdCopy(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "cp",
		Short:             `Copy stash resources from one namespace to another namespace`,
		DisableAutoGenTag: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := clientGetter.ToRESTConfig()
			if err != nil {
				return errors.Wrap(err, "failed to read kubeconfig")
			}

			srcNamespace, _, err = clientGetter.ToRawKubeConfigLoader().Namespace()

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

			vsClient, err = vs_cs.NewForConfig(cfg)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.AddCommand(NewCmdCopyRepository())
	cmd.AddCommand(NewCmdCopySecret())
	cmd.AddCommand(NewCmdCopyVolumeSnapshot())
	cmd.AddCommand(NewCmdCopyBackupConfiguration())

	cmd.PersistentFlags().StringVar(&dstNamespace, "to-namespace", dstNamespace, "Destination namespace.")
	return cmd
}
