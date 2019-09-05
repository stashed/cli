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
		stash clone pvc source-data -n demo --to-namespace=demo-1 --secret=gcs-secret --bucket=appscode-qa --prefix=/source/data --provider=gcs`)
)


func NewCmdClonePVC() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "pvc",
		Short:             `Clone PVC`,
		Long:              ``,
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

			// Cloning a PVC into a destination namespace, At first, need to create a Repository using backend credential
			// then take backup of the source PVC then restore the backed up data into VolumeClaimTemplate.
            repoName := fmt.Sprintf("%s-%s",repoOpt.provider, "repo")
			log.Infof("Creating Repository %s to %s namespace", repoName, dstNamespace)
			repository, err := createRepository(repoName, srcNamespace)
			if err != nil {
				return err
			}
			log.Infof("Repository %s/%s has been created successfully.", repository.Namespace, repository.Name)

			log.Infof("Repository are used to take backup of the source %s PVC\nTaking backup...", pvcName)

			err = backup(pvcName, repoName)
			if err != nil {
				return err
			}
			log.Infof("The PVC %s/%s data has been backed up successfully", pvc.Namespace, pvc.Name)

			log.Infof("Cloning PVC to new namespace...")
			err = clonePVC(pvc, repoName)
			if err != nil {
				return err
			}
			log.Infof("Cloning PVC has been performed successfully")

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

func backup(pvcName string, repoName string) error {
	// Take Backup
	configureBackupConfiguration(pvcName, repoName)
	backupConfig, err := createBackupConfiguration(fmt.Sprintf("%s-%s", pvcName, "backup"), srcNamespace)
	if err != nil {
		return err
	}
	log.Infof("BackupConfiguration %s/%s has been created successfully.", backupConfig.Namespace, backupConfig.Name)
	log.Infof("Waiting for backup success...")
	err = ensureBackup(backupConfig.Name, backupConfig.Namespace)
	if err != nil {
		return err
	}
	// Delete BackupConfiguration to stop taking backup
	return stashClient.StashV1beta1().BackupConfigurations(srcNamespace).Delete(backupConfig.Name, &metav1.DeleteOptions{})
}

func configureBackupConfiguration(pvcName string, repoName string)  {
	backupConfigOpt.task = "pvc-backup"
	backupConfigOpt.schedule = "*/1 * * * *"
	backupConfigOpt.repository = repoName
	backupConfigOpt.retentionPolicy.KeepLast = 5
	backupConfigOpt.retentionPolicy.Prune = true
	backupConfigOpt.targetRef = v1beta1.TargetRef{
		Name: pvcName,
		Kind: apis.KindPersistentVolumeClaim,
		APIVersion: core.SchemeGroupVersion.String(),
	}
	backupConfigOpt.volumeMounts = []string{"source-data:/source/data"}
	backupConfigOpt.paths = []string{"/source/data"}
}

func ensureBackup(name string, namespace string) error{
  return WaitUntilBackupSessionSucceed(name, namespace)
}

func clonePVC(pvc *core.PersistentVolumeClaim, repoName string) error {

	// create RestoreSession to create a new PVC in the destination namespace
	// then restore the backed up data into the PVC
	configureRestoreSession(pvc, repoName)

	restoreSession, err := createRestoreSession(fmt.Sprintf("%s-%s", pvc.Name, "restore"), srcNamespace)
	if err != nil {
		return err
	}
	log.Infof("RestoreSession %s/%s has been created successfully", restoreSession.Namespace, restoreSession.Name)
	log.Infof("Waiting for restore success...")
	err = ensureClonePVC(restoreSession.Name, restoreSession.Namespace)
	if err != nil {
		return err
	}

	// Delete Repository
	return stashClient.StashV1alpha1().Repositories(srcNamespace).Delete(repoName, &metav1.DeleteOptions{})
}

func configureRestoreSession(pvc *core.PersistentVolumeClaim, repoName string) {
	restoreSessionOpt.repository = repoName
	restoreSessionOpt.volumeClaimTemplate =  volumeclaimTemplate{
		accessModes: getPVCAccessModesAsStrings(pvc.Spec.AccessModes),
		storageClass: *pvc.Spec.StorageClassName,
		size: quantityTypePointer(pvc.Spec.Resources.Requests[core.ResourceStorage]).String(),
		name: "restore-data",
	}
	restoreSessionOpt.rule.Paths = []string{"/source/data"}
	restoreSessionOpt.volumeMounts = []string{"restore-data:/source/data"}
}

func ensureClonePVC(name string, namespace string) error {
	return WaitUntilRestoreSessionSucceed(name, namespace)
}

func getPVCAccessModesAsStrings(acMode []core.PersistentVolumeAccessMode) []string {
	accessModes := []string{}
	for _, am := range acMode {
		accessModes = append(accessModes, string(am))
	}
	return accessModes
}

