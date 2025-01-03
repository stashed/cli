package pkg

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/util"
)

type migrateOptions struct {
	kubeClient *kubernetes.Clientset
	config     *rest.Config
	repo       *v1alpha1.Repository
}

func NewCmdMigrateRepositoryToV2(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	opt := migrateOptions{}

	cmd := &cobra.Command{
		Use:               "migrate",
		Short:             `Migrate restic repository to v2`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("repository name not found")
			}
			repositoryName := args[0]

			var err error
			opt.config, err = clientGetter.ToRESTConfig()
			if err != nil {
				return errors.Wrap(err, "failed to read kubeconfig")
			}
			namespace, _, err = clientGetter.ToRawKubeConfigLoader().Namespace()
			if err != nil {
				return err
			}

			stshclient, err := cs.NewForConfig(opt.config)
			if err != nil {
				return err
			}

			opt.repo, err = stshclient.StashV1alpha1().Repositories(namespace).Get(context.TODO(), repositoryName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			opt.kubeClient, err = kubernetes.NewForConfig(opt.config)
			if err != nil {
				return err
			}

			if opt.repo.Spec.Backend.Local != nil {
				// get the pod that mount this repository as volume
				pod, err := getBackendMountingPod(opt.kubeClient, opt.repo)
				if err != nil {
					return err
				}
				return opt.migrateRepoFromPod(pod)
			}

			return opt.migrateRepo()
		},
	}

	cmd.Flags().StringVar(&imgRestic.Registry, "docker-registry", imgRestic.Registry, "Docker image registry for restic cli")
	cmd.Flags().StringVar(&imgRestic.Tag, "image-tag", imgRestic.Tag, "Restic docker image tag")

	return cmd
}

func (opt *migrateOptions) migrateRepoFromPod(pod *core.Pod) error {
	if err := opt.executeMigrateRepoCmdInPod(pod); err != nil {
		return err
	}

	klog.Infof("Repository %s/%s upgraded to version 2", namespace, opt.repo.Name)
	return nil
}

func (opt *migrateOptions) executeMigrateRepoCmdInPod(pod *core.Pod) error {
	command := []string{"/stash-enterprise", "migrate"}
	command = append(command, "--repo-name", opt.repo.Name, "--repo-namespace", opt.repo.Namespace)

	out, err := execCommandOnPod(opt.kubeClient, opt.config, pod, command)
	if string(out) != "" {
		klog.Infoln("Output:", string(out))
	}
	return err
}

func (opt *migrateOptions) migrateRepo() error {
	// get source repository secret
	secret, err := opt.kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), opt.repo.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if err = os.MkdirAll(ScratchDir, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(ScratchDir)

	// configure restic wrapper
	extraOpt := util.ExtraOptions{
		StorageSecret: secret,
		ScratchDir:    ScratchDir,
	}
	// configure setupOption
	setupOpt, err := util.SetupOptionsForRepository(*opt.repo, extraOpt)
	if err != nil {
		return fmt.Errorf("setup option for repository failed")
	}

	resticWrapper, err := restic.NewResticWrapper(setupOpt)
	if err != nil {
		return err
	}

	localDirs := &cliLocalDirectories{
		configDir: filepath.Join(ScratchDir, configDirName),
	}
	// dump restic's environments into `restic-env` file.
	// we will pass this env file to restic docker container.
	err = resticWrapper.DumpEnv(localDirs.configDir, ResticEnvs)
	if err != nil {
		return err
	}

	extraAgrs := []string{
		"upgrade_repo_v2", "--no-cache",
	}

	// For TLS secured Minio/REST server, specify cert path
	if resticWrapper.GetCaPath() != "" {
		extraAgrs = append(extraAgrs, "--cacert", resticWrapper.GetCaPath())
	}

	// run restore inside docker
	if err = runCmdViaDocker(*localDirs, "migrate", extraAgrs); err != nil {
		return err
	}
	klog.Infof("Repository %s/%s upgraded to version 2", namespace, opt.repo.Name)
	return nil
}
