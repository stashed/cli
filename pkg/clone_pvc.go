package pkg

import (
	"fmt"
	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubectl/util/templates"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
)

var (
	cloneExample = templates.Examples(`
		# Clone PVC
		stash clone pvc source -n demo --to-namespace=demo-1 --secret=<secret> --bucket=<bucket> --prefix=<prefix> --provider=<provider>`)
)

func NewCmdClonePVC() *cobra.Command {
	var repoOpt = repositoryOption{}
	var cmd = &cobra.Command{
		Use:               "pvc",
		Short:             `Clone PVC`,
		Long:              `Use Backup and Restore process for cloning PVC`,
		Example:           cloneExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("PVC name is not provided ")
			}

			pvcName := args[0]

			pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(srcNamespace).Get(pvcName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			// to clone a PVC from source namespace to destination namespace, we have to do the following steps:
			// create a Repository in the Source namespace.
			// create a BackupConfiguration to take backup of the source PVC in the same namespace.
			// then restore the backed up data into VolumeClaimTemplate in the destination namespace.
			repoName := fmt.Sprintf("%s-%s", repoOpt.provider, "repo")
			log.Infof("Creating Repository: %s to the Namespace: %s", repoName, srcNamespace)
			repository := getRepository(repoOpt, repoName, srcNamespace)
			repository, err = createRepository(repository)
			if err != nil {
				return err
			}
			log.Infof("Repository has been created successfully.")

			err = backupPVC(pvcName, repoName)
			if err != nil {
				return err
			}
			log.Infof("The PVC %s/%s data has been backed up successfully", pvc.Namespace, pvc.Name)

			// copy repository and secret to the destination namespace
			err = ensureRepository(repoName)
			if err != nil {
				return err
			}

			err = restorePVC(pvc, repoName)
			if err != nil {
				return err
			}
			// delete all repository
			err = cleanupRepository(repoName)
			if err != nil {

			}
			log.Infof("PVC has been cloned successfully!!")

			return nil
		},
	}
	cmd.Flags().StringVar(&repoOpt.provider, "provider", repoOpt.provider, "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&repoOpt.bucket, "bucket", repoOpt.bucket, "Name of the cloud bucket/container")
	cmd.Flags().StringVar(&repoOpt.endpoint, "endpoint", repoOpt.endpoint, "Endpoint for s3/s3 compatible backend")
	cmd.Flags().IntVar(&repoOpt.maxConnections, "max-connections", repoOpt.maxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")
	cmd.Flags().StringVar(&repoOpt.secret, "secret", repoOpt.secret, "Name of the Storage Secret")
	cmd.Flags().StringVar(&repoOpt.prefix, "prefix", repoOpt.prefix, "Prefix denotes the directory inside the backend")
	return cmd
}

// at first, create BackupConfiguration to take backup
// after successful taking backup, delete the BackupConfiguration to stop taking backup
func backupPVC(pvcName string, repoName string) error {
	// configure BackupConfiguration
	backupConfigOpt := backupConfigOption{
		task:       "pvc-backup",
		schedule:   "*/60 * * * *",
		repository: repoName,
		retentionPolicy: v1alpha1.RetentionPolicy{
			KeepLast: 5,
			Prune:    true,
		},
		targetRef: v1beta1.TargetRef{
			Name:       pvcName,
			Kind:       apis.KindPersistentVolumeClaim,
			APIVersion: core.SchemeGroupVersion.String(),
		},
	}
	backupConfig, err := getBackupConfiguration(backupConfigOpt, fmt.Sprintf("%s-%s", pvcName, "backup"), srcNamespace)
	if err != nil {
		return err
	}
	log.Infof("Creating BackupConfiguration: %s to the namespace: %s", backupConfig.Name, backupConfig.Namespace)
	backupConfig, err = createBackupConfiguration(backupConfig)
	if err != nil {
		return err
	}
	log.Infof("BackupConfiguration has been created successfully.")

	err = ensureInstantBackup(backupConfig, stashClient)
	if err != nil {
		return err
	}

	err = WaitUntilBackupSessionCompleted(backupConfig.Name, backupConfig.Namespace)
	if err != nil {
		return err
	}
	log.Infof("BackupSession has been succeeded.")
	// delete the BackupConfiguration to stop taking backup
	return stashClient.StashV1beta1().BackupConfigurations(srcNamespace).Delete(backupConfig.Name, &metav1.DeleteOptions{})
}

// create RestoreSession to create a new PVC in the destination namespace
// then restore the backed up data into the PVC
func restorePVC(pvc *core.PersistentVolumeClaim, repoName string) error {
	// configure RestoreSession
	restoreSessionOpt := restoreSessionOption{
		repository: repoName,
		task:       "pvc-restore",
		volumeClaimTemplate: volumeclaimTemplate{
			accessModes:  getPVCAccessModesAsStrings(pvc.Spec.AccessModes),
			storageClass: *pvc.Spec.StorageClassName,
			size:         getQuantityTypePointer(pvc.Spec.Resources.Requests[core.ResourceStorage]).String(),
			name:         pvc.Name,
		},
		rule: v1beta1.Rule{
			Snapshots: []string{"latest"},
		},
	}

	restoreSession, err := getRestoreSession(restoreSessionOpt, fmt.Sprintf("%s-%s", pvc.Name, "restore"), dstNamespace)
	if err != nil {
		return err
	}
	log.Infof("Creating RestoreSession: %s to the namespace: %s", restoreSession.Name, restoreSession.Namespace)
	restoreSession, err = createRestoreSession(restoreSession)
	if err != nil {
		return err
	}
	log.Infof("RestoreSession has been created successfully.")
	err = WaitUntilRestoreSessionCompleted(restoreSession.Name, restoreSession.Namespace)
	if err != nil {
		return err
	}
	log.Infof("RestoreSession has been succeeded.")
	// delete RestoreSession
	return stashClient.StashV1beta1().RestoreSessions(dstNamespace).Delete(restoreSession.Name, &metav1.DeleteOptions{})
}

func getPVCAccessModesAsStrings(acMode []core.PersistentVolumeAccessMode) []string {
	accessModes := []string{}
	for _, am := range acMode {
		accessModes = append(accessModes, string(am))
	}
	return accessModes
}

func cleanupRepository(repoName string) error {
	err := stashClient.StashV1alpha1().Repositories(srcNamespace).Delete(repoName, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return stashClient.StashV1alpha1().Repositories(dstNamespace).Delete(repoName, &metav1.DeleteOptions{})
}
