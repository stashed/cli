package pkg

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func NewCmdCreate(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "create",
		Short:             `Create stash resources `,
		DisableAutoGenTag: true,
	}
	//cmd.AddCommand(NewCmdCopyCreateRepo(clientGetter))
	return cmd
}
