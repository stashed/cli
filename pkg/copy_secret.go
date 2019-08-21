package pkg

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

func NewCmdCopySecret(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	var (
		toNamespace string
	)

	var cmd = &cobra.Command{
		Use:               "secret",
		Short:             `Copy Secret`,
		Long:              `Copy Secret from one namespace to another namespace`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("secret name not found")
			}

			secretName := args[0]

			cfg, err := clientGetter.ToRESTConfig()
			if err != nil {
				return errors.Wrap(err, "failed to read kubeconfig")
			}

			srcNamespace, _, err := clientGetter.ToRawKubeConfigLoader().Namespace()
			if err != nil {
				return err
			}

			kc, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				return err
			}

			// get source secret
			secret, err := kc.CoreV1().Secrets(srcNamespace).Get(secretName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// create/patch destination secret
			// only copy data
			err = createOrPatchSecretToNewNamespace(secret, toNamespace, kc)
			if err != nil {
				return err
			}

			log.Infof("Secret %s has been copied from namespace %s to %s successfully", secret.Name, srcNamespace, toNamespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&toNamespace, "to-namespace", toNamespace, "Destination namespace.")

	return cmd
}

// CreateOrPatch New Secret
func createOrPatchSecretToNewNamespace(secret *core.Secret, toNamespace string, kc kubernetes.Interface) error{
	_, _, err := core_util.CreateOrPatchSecret(
		kc,
		metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: toNamespace,
		},
		func(obj *core.Secret) *core.Secret {
			obj.Data = secret.Data
			return obj
		},
	)
	return err
}
