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
			// get source repository in current namespace
			// if found then copy the repository to destination namespace
			err := ensureRepository(repositoryName)

			return err
		},
	}

	return cmd
}

func ensureRepository(name string) error {
	// get source repository
	repository, err := stashClient.StashV1alpha1().Repositories(srcNamespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	log.Infof("Repository %s/%s uses Storage Secret %s/%s.", repository.Namespace, repository.Name, repository.Namespace, repository.Spec.Backend.StorageSecretName)
	// ensure source repository secret
	err = ensureSecret(repository.Spec.Backend.StorageSecretName)
	if err != nil {
		return err
	}
	err = copyRepository(repository)
	if err != nil {
		return err
	}
	log.Infof("Repository %s/%s has been copied to %s namespace successfully.", repository.Namespace, repository.Name, dstNamespace)
	return err
}

// CreateOrPatch New Secret
func copyRepository(repository *v1alpha1.Repository) error{
	_, _, err := util.CreateOrPatchRepository(
		stashClient.StashV1alpha1(),
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
