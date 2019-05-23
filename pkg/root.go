package pkg

import (
	"flag"

	"github.com/appscode/go/flags"
	v "github.com/appscode/go/version"
	"github.com/spf13/cobra"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"kmodules.xyz/client-go/logs"
	"kmodules.xyz/client-go/tools/cli"
	ocscheme "kmodules.xyz/openshift/client/clientset/versioned/scheme"
	stash_cli "stash.appscode.dev/cli/pkg/cli"
	"stash.appscode.dev/cli/pkg/docker"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/client/clientset/versioned/scheme"
	"stash.appscode.dev/stash/pkg/util"
)

func NewRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:               "stash",
		Short:             `kubectl plugin for Stash by AppsCode`,
		Long:              `kubectl plugin for Stash by AppsCode. For more information, visit here: https://appscode.com/products/stash`,
		DisableAutoGenTag: true,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			flags.DumpAll(c.Flags())
			cli.SendAnalytics(c, v.Version.Version)

			scheme.AddToScheme(clientsetscheme.Scheme)
			ocscheme.AddToScheme(clientsetscheme.Scheme)
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	logs.ParseFlags()
	rootCmd.PersistentFlags().StringVar(&util.ServiceName, "service-name", "stash-operator", "Stash service name.")
	rootCmd.PersistentFlags().BoolVar(&cli.EnableAnalytics, "enable-analytics", cli.EnableAnalytics, "Send analytical events to Google Analytics")
	rootCmd.PersistentFlags().BoolVar(&apis.EnableStatusSubresource, "enable-status-subresource", apis.EnableStatusSubresource, "If true, uses sub resource for crds.")

	rootCmd.AddCommand(v.NewCmdVersion())

	rootCmd.AddCommand(stash_cli.NewCopyRepositoryCmd())
	rootCmd.AddCommand(stash_cli.NewUnlockRepositoryCmd())
	rootCmd.AddCommand(stash_cli.NewUnlockLocalRepositoryCmd())
	rootCmd.AddCommand(stash_cli.NewTriggerBackupCmd())
	rootCmd.AddCommand(stash_cli.NewBackupPVCmd())
	rootCmd.AddCommand(stash_cli.NewDownloadCmd())
	rootCmd.AddCommand(stash_cli.NewDeleteSnapshotCmd())

	rootCmd.AddCommand(docker.NewUnlockRepositoryCmd())
	rootCmd.AddCommand(docker.NewDownloadCmd())
	rootCmd.AddCommand(docker.NewDeleteSnapshotCmd())

	return rootCmd
}
