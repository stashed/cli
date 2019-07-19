package pkg

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	core_util "kmodules.xyz/client-go/core/v1"
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

			// for local backend create/patch PVC
			if repository.Spec.Backend.Local != nil && repository.Spec.Backend.Local.PersistentVolumeClaim != nil {
				// get PVC
				pvc, err := kc.CoreV1().PersistentVolumeClaims(srcNamespace).Get(
					repository.Spec.Backend.Local.PersistentVolumeClaim.ClaimName,
					metav1.GetOptions{},
				)
				if err != nil {
					return err
				}
				_, _, err = core_util.CreateOrPatchPVC(
					kc,
					metav1.ObjectMeta{
						Name:      pvc.Name,
						Namespace: toNamespace,
					},
					func(obj *core.PersistentVolumeClaim) *core.PersistentVolumeClaim {
						obj.Spec = pvc.Spec
						return obj
					},
				)
				if err != nil {
					return err
				}
				log.Infof("PVC %s copied from namespace %s to %s", pvc.Name, srcNamespace, toNamespace)
			}

			// create/patch destination repository secret
			// only copy data
			_, _, err = core_util.CreateOrPatchSecret(
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
			if err != nil {
				return err
			}
			log.Infof("Secret %s copied from namespace %s to %s", secret.Name, srcNamespace, toNamespace)

			// create/patch destination repository
			// only copy spec
			_, _, err = util.CreateOrPatchRepository(
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
			if err != nil {
				return err
			}
			log.Infof("Repository %s copied from namespace %s to %s", repositoryName, srcNamespace, toNamespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&toNamespace, "to-namespace", toNamespace, "Destination namespace.")

	return cmd
}
