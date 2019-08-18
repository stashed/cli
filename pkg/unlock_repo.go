package pkg

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"stash.appscode.dev/stash/pkg/restic"

	"github.com/appscode/go/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"stash.appscode.dev/cli/pkg/docker"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/util"
)

func NewCmdUnlockRepositoryCrd(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	var (
		localDirs = &cliLocalDirectories{}
	)
	var cmd = &cobra.Command{
		Use:               "unlock_repo",
		Short:             `Unlock Restic Repository`,
		Long:              `Unlock Restic Repository`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("Repository name not found")
			}
			repositoryName := args[0]

			cfg, err := clientGetter.ToRESTConfig()
			if err != nil {
				return errors.Wrap(err, "failed to read kubeconfig")
			}
			namespace, _, err := clientGetter.ToRawKubeConfigLoader().Namespace()
			if err != nil {
				return err
			}

			kc, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				return err
			}
			client, err := cs.NewForConfig(cfg)
			if err != nil {
				return err
			}

			// get source repository
			repository, err := client.StashV1alpha1().Repositories(namespace).Get(repositoryName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// get source repository secret
			secret, err := kc.CoreV1().Secrets(namespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// write secret in a temp dir and
			// cleanup whole tempDir dir at the end
			tempDir := filepath.Join("/tmp")
			defer os.RemoveAll(docker.SecretDir)

			// prepare local dirs
			if err = localDirs.prepareDir(tempDir, secret); err != nil {
				return err
			}

			// configure restic wrapper
			extraOpt := util.ExtraOptions{
				SecretDir:   docker.SecretDir,
				EnableCache: false,
				ScratchDir:  docker.ScratchDir,
			}
			// configure setupOption
			setupOpt, err := util.SetupOptionsForRepository(*repository, extraOpt)
			if err != nil {
				return fmt.Errorf("setup option for repository failed")
			}

			// init restic wrapper
			resticWrapper, err := restic.NewResticWrapper(setupOpt)
			if err != nil {
				return err
			}

			// run unlock inside docker
			if err = runUnlockRepoViaDocker(*localDirs, resticWrapper.GetRepo()); err != nil {
				return err
			}
			log.Infof("Repository %s/%s unlocked", namespace, repositoryName)
			return nil
		},
	}

	return cmd
}

func runUnlockRepoViaDocker(localDirs cliLocalDirectories, resticRepo string) error {
	// get current user
	currentUser, err := user.Current()
	if err != nil {
		return err
	}
	args := []string{
		"run",
		"--rm",
		"-u", currentUser.Uid,
		"-v", localDirs.secretDir + ":" + docker.SecretDir,
		"--env", "HTTP_PROXY=" + os.Getenv("HTTP_PROXY"),
		"--env", "HTTPS_PROXY=" + os.Getenv("HTTPS_PROXY"),
		"--env-file", filepath.Join(localDirs.secretDir, "env"),
		imgRestic.ToContainerImage(),
		"unlock",
		"--no-cache",
		"-r", resticRepo,
	}
	log.Infoln("Running docker with args:", args)
	out, err := exec.Command("docker", args...).CombinedOutput()
	log.Infoln("Output:", string(out))
	return err
}
