package pkg

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/reference"
	core_util "kmodules.xyz/client-go/core/v1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	stash_scheme "stash.appscode.dev/stash/client/clientset/versioned/scheme"
	"stash.appscode.dev/stash/pkg/util"
)

func NewCmdTriggerBackup(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "trigger",
		Short:             `Trigger a backup`,
		Long:              `Trigger a backup by creating BackupSession`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("BackupConfiguration name not found")
			}
			backupConfigName := args[0]

			cfg, err := clientGetter.ToRESTConfig()
			if err != nil {
				return errors.Wrap(err, "failed to read kubeconfig")
			}
			namespace, _, err := clientGetter.ToRawKubeConfigLoader().Namespace()
			if err != nil {
				return err
			}

			client, err := cs.NewForConfig(cfg)
			if err != nil {
				return err
			}

			// get backupConfiguration
			backupConfig, err := client.StashV1beta1().BackupConfigurations(namespace).Get(backupConfigName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			_, err = triggerBackup(backupConfig, client)
			return err
		},
	}

	return cmd
}

func triggerBackup(backupConfig *v1beta1.BackupConfiguration, client cs.Interface) (*v1beta1.BackupSession, error) {

	// create backupSession for backupConfig
	backupSession := &v1beta1.BackupSession{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: backupConfig.Name + "-",
			Namespace:    backupConfig.Namespace,
			Labels: map[string]string{
				util.LabelApp:                 util.AppLabelStash,
				util.LabelBackupConfiguration: backupConfig.Name,
			},
		},
		Spec: v1beta1.BackupSessionSpec{
			Invoker: v1beta1.BackupInvokerRef{
				APIGroup: v1beta1.SchemeGroupVersion.Group,
				Kind:     v1beta1.ResourceKindBackupConfiguration,
				Name:     backupConfig.Name,
			},
		},
	}

	// set backupConfig as backupSession's owner
	ref, err := reference.GetReference(stash_scheme.Scheme, backupConfig)
	if err != nil {
		return backupSession, err
	}
	core_util.EnsureOwnerReference(&backupSession.ObjectMeta, ref)

	// don't use createOrPatch here
	backupSession, err = client.StashV1beta1().BackupSessions(backupSession.Namespace).Create(backupSession)
	if err != nil {
		return backupSession, err
	}
	log.Infof("BackupSession %s/%s has been created successfully", backupSession.Namespace, backupSession.Name)
	return backupSession, nil
}
