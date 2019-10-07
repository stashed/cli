package pkg

import (
	vs_cs "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
)

var (
	dstNamespace string
	srcNamespace string
	kubeClient   *kubernetes.Clientset
	stashClient  *cs.Clientset
	vsClient     *vs_cs.Clientset
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
