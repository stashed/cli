package pkg

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"k8s.io/api/core/v1"
)

func NewCmdCopySecret() *cobra.Command {
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

			// get source secret
			secret, err := getSecret(srcNamespace, secretName)
			if err != nil {
				return err
			}

			// copy the secret to destination namespace
			err = copySecret(secret)
			if err != nil {
				return err
			}

			log.Infof("Secret %s/%s has been copied to %s namespace successfully.", srcNamespace,secret.Name, dstNamespace)
			return nil
		},
	}

	return cmd
}

// CreateOrPatch New Secret
func copySecret(secret *core.Secret) error{
	_, _, err := core_util.CreateOrPatchSecret(
		kc,
		metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: dstNamespace,
		},
		func(obj *core.Secret) *core.Secret {
			obj.Data = secret.Data
			return obj
		},
	)
	return err
}

func getSecret(namespace string, name string) (secret *v1.Secret, err error) {
	return kc.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
}
