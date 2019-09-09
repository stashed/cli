package pkg

import (
	"fmt"
	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubectl/util/templates"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
)

var (

	cloneExample = templates.Examples(`
		# Clone PVC
		stash clone pvc source -n demo --to-namespace=demo-1 --secret=<secret> --bucket=<bucket> --prefix=<prefix> --provider=<provider>`)
)


func NewCmdClonePVC() *cobra.Command {
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

			// cloning a PVC into a destination namespace, At first, need to create a Repository
			// and BackupConfiguration to take backup of the source PVC in the source namespace
			// then restore the backed up data into VolumeClaimTemplate in the destination namespace.
            repoName := fmt.Sprintf("%s-%s",repoOpt.provider, "repo")
			log.Infof("Creating Repository to source Namespace....")
			repository, err := createRepository(repoName, srcNamespace)
			if err != nil {
				return err
			}
			log.Infof("Repository %s/%s has been created successfully.", repository.Namespace, repository.Name)

			log.Infof("Repository are used by the BackupConfiguration to take backup of the source %s PVC\nStart backup process...", pvcName)
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

			log.Infof("Start Cloning Process...")
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
// then after successful taking backup, delete the BackupConfiguration to stop taking backup
func backupPVC(pvcName string, repoName string) error {
	// configure the BackupConfiguration
	setBackupConfiguration(pvcName, repoName)

	backupConfig, err := createBackupConfiguration(fmt.Sprintf("%s-%s", pvcName, "backup"), srcNamespace)
	if err != nil {
		return err
	}
	log.Infof("BackupConfiguration %s/%s has been created successfully.", backupConfig.Namespace, backupConfig.Name)
	log.Infof("Waiting for backup success...")
	err = WaitUntilBackupSessionSucceed(backupConfig.Name, backupConfig.Namespace)
	if err != nil {
		return err
	}
	// delete the BackupConfiguration to stop taking backup
	return stashClient.StashV1beta1().BackupConfigurations(srcNamespace).Delete(backupConfig.Name, &metav1.DeleteOptions{})
}

func setBackupConfiguration(pvcName string, repoName string)  {
	backupConfigOpt.task = "pvc-backup"
	backupConfigOpt.schedule = "*/3 * * * *"
	backupConfigOpt.repository = repoName
	backupConfigOpt.retentionPolicy.KeepLast = 5
	backupConfigOpt.retentionPolicy.Prune = true
	backupConfigOpt.targetRef = v1beta1.TargetRef{
		Name: pvcName,
		Kind: apis.KindPersistentVolumeClaim,
		APIVersion: core.SchemeGroupVersion.String(),
	}
}

// create RestoreSession to create a new PVC in the destination namespace
// then restore the backed up data into the PVC
func restorePVC(pvc *core.PersistentVolumeClaim, repoName string) error {
	// configure the RestoreSession
	setRestoreSession(pvc, repoName)

	restoreSession, err := createRestoreSession(fmt.Sprintf("%s-%s", pvc.Name, "restore"), dstNamespace)
	if err != nil {
		return err
	}
	log.Infof("RestoreSession %s/%s has been created successfully", restoreSession.Namespace, restoreSession.Name)
	log.Infof("Waiting for restore success...")
	err = WaitUntilRestoreSessionSucceed(restoreSession.Name, restoreSession.Namespace)
	if err != nil {
		return err
	}

	// delete RestoreSession
	return stashClient.StashV1beta1().RestoreSessions(dstNamespace).Delete(restoreSession.Name, &metav1.DeleteOptions{})
}

func setRestoreSession(pvc *core.PersistentVolumeClaim, repoName string) {
	restoreSessionOpt.repository = repoName
	restoreSessionOpt.task = "pvc-restore"
	restoreSessionOpt.volumeClaimTemplate =  volumeclaimTemplate{
		accessModes: getPVCAccessModesAsStrings(pvc.Spec.AccessModes),
		storageClass: *pvc.Spec.StorageClassName,
		size: getQuantityTypePointer(pvc.Spec.Resources.Requests[core.ResourceStorage]).String(),
		name: pvc.Name,
	}
	restoreSessionOpt.rule.Snapshots = []string{"latest"}
	namespace = dstNamespace
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