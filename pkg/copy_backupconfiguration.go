package pkg

import (
	"fmt"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	stash_v1beta1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"

	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
)

func NewCmdCopyBackupConfiguration() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "bc",
		Short:             `Copy BackupConfiguration`,
		Long:              `Copy BackupConfiguration from one namespace to another namespace`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("backupconfiguration name not found")
			}

			bcName := args[0]

			// get resource backupconfiguration
			bc, err := client.StashV1beta1().BackupConfigurations(srcNamespace).Get(bcName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if bc.Spec.Driver != v1beta1.VolumeSnapshotter {
				// get source repository using current namespace
				repository, err := getRepository(srcNamespace, bc.Spec.Repository.Name)
				if err != nil {
					return err
				}

				// get source repository secret using current namespace
				secret, err := getSecret(srcNamespace, repository.Spec.Backend.StorageSecretName)
				if err != nil {
					return err
				}

				// try to get secret in destination namespace
				// if not found, create new one
				_ , err = getSecret(dstNamespace, repository.Spec.Backend.StorageSecretName)
				if err != nil {
					if kerr.IsNotFound(err) {
						log.Infof("Repository %s/%s uses Storage Secret %s/%s.\nCopying Storage Secret %s to %s namespace", repository.Namespace, repository.Name,secret.Namespace, secret.Name, srcNamespace, dstNamespace)
						// copy the secret to destination namespace
						err = copySecret(secret)
						if err != nil {
							return err
						}
						log.Infof("Secret %s/%s has been copied to %s namespace successfully.", srcNamespace, secret.Name, dstNamespace)

					} else {
						return err
					}
				}

				// try to get repository in destination namespace
				// if not found, create new one
				_, err = getRepository(dstNamespace, bc.Spec.Repository.Name)
				if err != nil {
					if kerr.IsNotFound(err) {
						log.Infof("BackupConfiguration %s/%s uses Repository %s/%s.\nCopying Repository %s to %s namespace", bc.Namespace, bc.Name, repository.Namespace, repository.Name, srcNamespace, dstNamespace)
						// copy the repository to destination namespace
						err = copyRepository(repository)
						if err != nil {
							return err
						}
						log.Infof("Repository %s/%s has been copied to %s namespace successfully.", srcNamespace, repository.Name, dstNamespace)

					}else {
						return err
					}
				}
			}
            // copy the backupconfiguration to new namespace
            err = copyBackupConfiguration(bc)
            if err != nil {
            	return err
			}

			log.Infof("BackupConfiguration %s/%s has been copied to %s namespace successfully.", srcNamespace, bc.Name, dstNamespace)
			return nil
		},
	}

	return cmd
}

func  copyBackupConfiguration(bc *v1beta1.BackupConfiguration) error {

	_, _ , err := stash_v1beta1_util.CreateOrPatchBackupConfiguration(
		client.StashV1beta1(),
		metav1.ObjectMeta{
			Name: bc.Name,
			Namespace: dstNamespace,
		},
		func(obj *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
			obj.Spec = bc.Spec
			return obj
		},
	)
	return err
}


