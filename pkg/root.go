package pkg

import (
	"flag"

	v "github.com/appscode/go/version"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	cliflag "k8s.io/component-base/cli/flag"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"kmodules.xyz/client-go/logs"
	"kmodules.xyz/client-go/tools/cli"
	ocscheme "kmodules.xyz/openshift/client/clientset/versioned/scheme"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/client/clientset/versioned/scheme"
)

func NewRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:               "stash",
		Short:             `kubectl plugin for Stash by AppsCode`,
		Long:              `kubectl plugin for Stash by AppsCode. For more information, visit here: https://appscode.com/products/stash`,
		DisableAutoGenTag: true,
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			cli.SendAnalytics(c, v.Version.Version)

			err := scheme.AddToScheme(clientsetscheme.Scheme)
			if err != nil {
				return err
			}
			return ocscheme.AddToScheme(clientsetscheme.Scheme)
		},
	}

	flags := rootCmd.PersistentFlags()
	// Normalize all flags that are coming from other packages or pre-configurations
	// a.k.a. change all "_" to "-". e.g. glog package
	flags.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)

	kubeConfigFlags := genericclioptions.NewConfigFlags(true)
	kubeConfigFlags.AddFlags(flags)
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	matchVersionKubeConfigFlags.AddFlags(flags)

	flags.AddGoFlagSet(flag.CommandLine)
	logs.ParseFlags()
	flags.BoolVar(&cli.EnableAnalytics, "enable-analytics", cli.EnableAnalytics, "Send analytical events to Google Analytics")
	flags.BoolVar(&apis.EnableStatusSubresource, "enable-status-subresource", apis.EnableStatusSubresource, "If true, uses sub resource for crds.")

	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	rootCmd.AddCommand(v.NewCmdVersion())

	rootCmd.AddCommand(NewCmdCopy(f))
	rootCmd.AddCommand(NewCmdDelete(f))
	rootCmd.AddCommand(NewCmdDownloadRepository(f))
	rootCmd.AddCommand(NewCmdTriggerBackup(f))
	rootCmd.AddCommand(NewCmdUnlockRepository(f))
	rootCmd.AddCommand(NewCmdUnlockRepositoryCrd(f))

	return rootCmd
}
