package pkg

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
)

func NewCmdCopyRepository(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	var (
		toNamespace string
	)

	var cmd = &cobra.Command{
		Use:               "repository",
		Short:             `Copy Repository and Secret`,
		Long:              `Copy Repository and Secret from one namespace to another namespace`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("Repository name not found")
			}

			repositoryName := args[0]

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
			client, err := cs.NewForConfig(cfg)
			if err != nil {
				return err
			}
			// get source repository
			repository, err := client.StashV1alpha1().Repositories(srcNamespace).Get(repositoryName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			// get source repository secret
			secret, err := kc.CoreV1().Secrets(srcNamespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// create/patch destination repository secret
			// only copy data
			err = createOrPatchSecretToNewNamespace(secret, toNamespace, kc)
			if err != nil {
				return err
			}
			log.Infof("Secret %s has been copied from namespace %s to %s successfully for %s repository", secret.Name, srcNamespace, toNamespace, repository.Name)

			// create/patch destination repository
			// only copy spec
			err = createOrPatchRepositortToNewNamespace(repository, toNamespace, client)
			if err != nil {
				return err
			}
			if err != nil {
				return err
			}
			log.Infof("Repository %s has been copied from namespace %s to %s successfully", repositoryName, srcNamespace, toNamespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&toNamespace, "to-namespace", toNamespace, "Destination namespace.")

	return cmd
}

// CreateOrPatch New Secret
func createOrPatchRepositortToNewNamespace(repository *v1alpha1.Repository, toNamespace string, client cs.Interface) error{
	_, _, err := util.CreateOrPatchRepository(
		client.StashV1alpha1(),
		metav1.ObjectMeta{
			Name:      repository.Name,
			Namespace: toNamespace,
		},
		func(obj *v1alpha1.Repository) *v1alpha1.Repository {
			obj.Spec = repository.Spec
			return obj
		},
	)
	return err
}

