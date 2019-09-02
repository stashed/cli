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
				return fmt.Errorf("BackupConfiguration name is not provided")
			}

			backupConfigName := args[0]
			// get source BackupConfiguration and respective Repository and Secret in current namespace
			// if found then copy the BackupConfiguration, Repository and Secret to destination namespace
			return  ensureBackupConfiguration(backupConfigName)
		},
	}

	return cmd
}

func ensureBackupConfiguration(name string) error {
	// get resource BackupConfiguration
	backupConfig, err := stashClient.StashV1beta1().BackupConfigurations(srcNamespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}
    // Repository holds the backend information, In Restic driver mechanism, Repository is used to backup.
    // For that need to insure Repository and Secret
	if backupConfig.Spec.Driver != v1beta1.VolumeSnapshotter {
		// ensure Repository and Secret
		err = ensureRepository(backupConfig.Spec.Repository.Name)
		if err != nil {
			return err
		}
	}

	err = copyBackupConfiguration(backupConfig)
	if err != nil {
		return err
	}

	log.Infof("BackupConfiguration %s/%s has been copied to %s namespace successfully.", srcNamespace, backupConfig.Name, dstNamespace)
	return err
}

func  copyBackupConfiguration(bc *v1beta1.BackupConfiguration) error {
	meta := metav1.ObjectMeta{
		Name: bc.Name,
		Namespace: dstNamespace,
	}
	_, _ , err := v1beta1_util.CreateOrPatchBackupConfiguration(
		stashClient.StashV1beta1(),
		meta,
		func(in *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
			in.Spec = bc.Spec
			return in
		},
	)
	return err
}


