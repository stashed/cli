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
	"stash.appscode.dev/apimachinery/client/clientset/versioned/scheme"

	"github.com/spf13/cobra"
	v "gomodules.xyz/x/version"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	ocscheme "kmodules.xyz/openshift/client/clientset/versioned/scheme"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               "kubectl-stash",
		Short:             `kubectl plugin for Stash by AppsCode`,
		Long:              `kubectl plugin for Stash by AppsCode. For more information, visit here: https://appscode.com/products/stash`,
		DisableAutoGenTag: true,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			utilruntime.Must(scheme.AddToScheme(clientsetscheme.Scheme))
			utilruntime.Must(ocscheme.AddToScheme(clientsetscheme.Scheme))
		},
	}

	flags := rootCmd.PersistentFlags()

	kubeConfigFlags := genericclioptions.NewConfigFlags(true)
	kubeConfigFlags.AddFlags(flags)
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	matchVersionKubeConfigFlags.AddFlags(flags)

	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	rootCmd.AddCommand(v.NewCmdVersion())
	rootCmd.AddCommand(NewCmdCompletion())

	rootCmd.AddCommand(NewCmdCopy(f))
	rootCmd.AddCommand(NewCmdDelete(f))
	rootCmd.AddCommand(NewCmdDownloadRepository(f))
	rootCmd.AddCommand(NewCmdTriggerBackup(f))
	rootCmd.AddCommand(NewCmdUnlockRepository(f))
	rootCmd.AddCommand(NewCmdCreate(f))
	rootCmd.AddCommand(NewCmdClone(f))
	rootCmd.AddCommand(NewCmdPause(f))
	rootCmd.AddCommand(NewCmdResume(f))
	rootCmd.AddCommand(NewCmdDebug(f))
	return rootCmd
}
