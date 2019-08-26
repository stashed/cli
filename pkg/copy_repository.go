package pkg

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
)

func NewCmdCopyRepository() *cobra.Command {

	var cmd = &cobra.Command{
		Use:               "repository",
		Short:             `Copy Repository and Secret`,
		Long:              `Copy Repository and Secret from one namespace to another namespace`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("repository name not found")
			}

			repositoryName := args[0]

			// get source repository
			repository, err := getRepository(srcNamespace, repositoryName)
			if err != nil {
				return err
			}

			// get source repository secret
			secret, err := getSecret(srcNamespace, repository.Spec.Backend.StorageSecretName)
			if err != nil {
				return err
			}

			log.Infof("Repository %s/%s uses Storage Secret %s/%s.\nCopying Storage Secret %s to %s namespace", repository.Namespace, repository.Name,secret.Namespace, secret.Name, srcNamespace, dstNamespace)
			// copy the secret to destination namespace
			err = copySecret(secret)
			if err != nil {
				return err
			}
			log.Infof("Secret %s/%s has been copied to %s namespace successfully.", srcNamespace, secret.Name, dstNamespace)

			// copy the repository to destination namespace
			err = copyRepository(repository)
			if err != nil {
				return err
			}
			log.Infof("Repository %s/%s has been copied to %s namespace successfully.", srcNamespace, repositoryName, dstNamespace)
			return nil
		},
	}

	return cmd
}

// CreateOrPatch New Secret
func copyRepository(repository *v1alpha1.Repository) error{
	_, _, err := util.CreateOrPatchRepository(
		client.StashV1alpha1(),
		metav1.ObjectMeta{
			Name:      repository.Name,
			Namespace: dstNamespace,
		},
		func(obj *v1alpha1.Repository) *v1alpha1.Repository {
			obj.Spec = repository.Spec
			return obj
		},
	)
	return err
}


func getRepository(namespace string, name string) (repository *v1alpha1.Repository, err error) {
	return client.StashV1alpha1().Repositories(namespace).Get(name, metav1.GetOptions{})
}
