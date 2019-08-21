package pkg

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func NewCmdCopy(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "cp",
		Short:             `Copy stash resources from one namespace to another namespace`,
		DisableAutoGenTag: true,
	}
	cmd.AddCommand(NewCmdCopyRepository(clientGetter))
	cmd.AddCommand(NewCmdCopySecret(clientGetter))
	return cmd
}
