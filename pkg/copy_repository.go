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
				return fmt.Errorf("Repository name is not provided")
			}

			repositoryName := args[0]
			// get source Repository in current namespace
			// if found then copy the Repository to destination namespace
			return  ensureRepository(repositoryName)
		},
	}

	return cmd
}

func ensureRepository(name string) error {
	// get source Repository
	repository, err := stashClient.StashV1alpha1().Repositories(srcNamespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	log.Infof("Repository %s/%s uses Storage Secret %s/%s.", repository.Namespace, repository.Name, repository.Namespace, repository.Spec.Backend.StorageSecretName)
	// ensure source Repository Secret
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
	meta := metav1.ObjectMeta{
		Name:      repository.Name,
		Namespace: dstNamespace,
	}
	_, _, err := util.CreateOrPatchRepository(stashClient.StashV1alpha1(), meta, func(in *v1alpha1.Repository) *v1alpha1.Repository {
			in.Spec = repository.Spec
			return in
		},
	)
	return err
}
