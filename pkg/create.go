package pkg

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	core "k8s.io/api/core/v1"
)

var (
	namespace   string
	kubeClient  *kubernetes.Clientset
	stashClient *cs.Clientset

	opt                 = option{}
	volumeMounts        []core.VolumeMount
	paths               []string
	snapshots            []string
	volumeClaimTemplate core.PersistentVolumeClaim
)

type option struct {
	// For Repository
	Provider       string
	Bucket         string
	Endpoint       string
	URL            string
	MaxConnections int
	Secret         string
	Prefix         string
	Container      string
    // For BackupConfiguration && RestoreSession
	Paths     string
	VolMounts string

	TargetRef       v1beta1.TargetRef
	RetentionPolicy v1alpha1.RetentionPolicy
	RepositoryName  string
	Schedule        string
	Driver          string
	VSClassName     string
	TaskName        string
	Replica         int32
	Pause           bool

	Rule            rule
	VolumeClaimTemp volumeclaimTemp
}

type volumeclaimTemp struct {
	Name             string
	AccessMode       string
	StorageClassName string
	Size             string
	DataSourceName   string
}

type rule struct {
	SourceHost string
	Snapshot  string
	Paths      string
}

func NewCmdCreate(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "create",
		Short:             `create stash resources`,
		DisableAutoGenTag: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

			cfg, err := clientGetter.ToRESTConfig()
			if err != nil {
				return errors.Wrap(err, "failed to read kubeconfig")
			}

			namespace, _, err = clientGetter.ToRawKubeConfigLoader().Namespace()
			if err != nil {
				return err
			}

			kubeClient, err = kubernetes.NewForConfig(cfg)
			if err != nil {
				return err
			}

			stashClient, err = cs.NewForConfig(cfg)
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmd.AddCommand(NewCmdCreateRepository())
	cmd.AddCommand(NewCmdCreateBackupConfiguration())
	cmd.AddCommand(NewCmdCreateRestoreSession())

	return cmd
}
