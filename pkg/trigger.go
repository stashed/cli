package pkg

import (
	"fmt"
	"time"

	"github.com/appscode/go/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
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

			// create backupSession for backupConfig
			backupSession := &v1beta1.BackupSession{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("%s-%d", backupConfigName, time.Now().Unix()),
					Namespace:    namespace,
					Labels: map[string]string{
						util.LabelApp:                 util.AppLabelStash,
						util.LabelBackupConfiguration: backupConfigName,
					},
				},
				Spec: v1beta1.BackupSessionSpec{
					BackupConfiguration: v1.LocalObjectReference{
						Name: backupConfigName,
					},
				},
			}

			// set backupConfig as backupSession's owner
			ref, err := reference.GetReference(stash_scheme.Scheme, backupConfig)
			if err != nil {
				return err
			}
			core_util.EnsureOwnerReference(&backupSession.ObjectMeta, ref)

			// don't use createOrPatch here
			backupSession, err = client.StashV1beta1().BackupSessions(namespace).Create(backupSession)
			if err != nil {
				return err
			}

			log.Infof("BackupConfiguration %s/%s has been triggered successfully by BackupSession %s/%s", backupConfig.Namespace, backupConfig.Name, backupSession.Namespace, backupSession.Name)
			return nil
		},
	}

	return cmd
}