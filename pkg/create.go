package pkg

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
)

var (
	namespace   string
	kubeClient  *kubernetes.Clientset
	stashClient *cs.Clientset
)

func NewCmdCreate(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "create",
		Short:             `create stash resources`,
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

			return nil
		},
	}
	cmd.AddCommand(NewCmdCreateRepository())
	cmd.AddCommand(NewCmdCreateBackupConfiguration())
	cmd.AddCommand(NewCmdCreateRestoreSession())

	return cmd
}
