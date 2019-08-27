package pkg

import (
	"fmt"
	"github.com/appscode/go/log"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	v1beta1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCmdCopyBackupConfiguration() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "backupconfig",
		Short:             `Copy BackupConfiguration from one namespace to another namespace`,
		Long:              `Copy BackupConfiguration with respective Repository and Secret if they are not present in the target namespace`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("backupconfiguration name not found")
			}

			backupConfigName := args[0]
			// get source backupconfiguration and respective repository and secret in current namespace
			// if found then Copy the BackupConfiguration, repository and secret to destination namespace
			err := ensureBackupConfiguration(backupConfigName)

			return err
		},
	}

	return cmd
}

func ensureBackupConfiguration(name string) error {
	// get resource backupconfiguration
	backupConfig, err := stashClient.StashV1beta1().BackupConfigurations(srcNamespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if backupConfig.Spec.Driver != v1beta1.VolumeSnapshotter {
		// get source repository
		repository, err := stashClient.StashV1alpha1().Repositories(backupConfig.Namespace).Get(backupConfig.Spec.Repository.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// get source repository
		_, err = kubeClient.CoreV1().Secrets(backupConfig.Namespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// ensure repository and secret
		err = ensureRepository(repository.Name)
	}

	err = copyBackupConfiguration(backupConfig)
	if err != nil {
		return err
	}

	log.Infof("BackupConfiguration %s/%s has been copied to %s namespace successfully.", srcNamespace, backupConfig.Name, dstNamespace)
	return err
}

func  copyBackupConfiguration(bc *v1beta1.BackupConfiguration) error {

	_, _ , err := v1beta1_util.CreateOrPatchBackupConfiguration(
		stashClient.StashV1beta1(),
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


