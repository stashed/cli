package pkg

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func NewCmdDelete(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete",
		Short:             `Delete stash resources`,
		DisableAutoGenTag: true,
	}
	cmd.AddCommand(NewCmdDeleteSnapshot(clientGetter))
	return cmd
}
