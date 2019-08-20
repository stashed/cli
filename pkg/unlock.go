package pkg

import (
	"fmt"
	"io/ioutil"
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
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/util"

)

func NewCmdUnlockRepository(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	var (
		localDirs = &cliLocalDirectories{}
	)
	var cmd = &cobra.Command{
		Use:               "unlock",
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

			tempDir, err := ioutil.TempDir("", "stash-cli")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tempDir)

			// dump secret into a temporary directory.
			// we will pass the secret files into restic docker container.
			if err = localDirs.dumpSecret(tempDir, secret); err != nil {
				return err
			}

			// configure restic wrapper
			extraOpt := util.ExtraOptions{
				SecretDir:   localDirs.secretDir,
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

			localDirs.configDir = filepath.Join(tempDir, configDirName)
			// dump restic's environments into `restic-env` file.
			// we will pass this env file to restic docker container.
			err = resticWrapper.DumpEnv(localDirs.configDir, ResticEnvs)
			if err != nil{
				return err
			}


			extraAgrs := []string{
				"--no-cache",
			}

			// For TLS secured Minio/REST server, specify cert path
			if _, err := os.Stat(filepath.Join(localDirs.secretDir, restic.CA_CERT_DATA)); err == nil {
				extraAgrs = append(extraAgrs, "--cacert", filepath.Join(localDirs.secretDir, restic.CA_CERT_DATA))
			}

			// run unlock inside docker
			if err = runCmdViaDocker(*localDirs, "unlock", extraAgrs); err != nil {
				return err
			}
			log.Infof("Repository %s/%s has been unlocked successfully", namespace, repositoryName)
			return nil
		},
	}

	return cmd
}

func runCmdViaDocker(localDirs cliLocalDirectories, command string, extraArgs []string) error {
	// get current user
	currentUser, err := user.Current()
	if err != nil {
		return err
	}
	args := []string{
		"run",
		"--rm",
		"-u", currentUser.Uid,
		"-v", localDirs.secretDir + ":" + localDirs.secretDir,
		"--env", "HTTP_PROXY=" + os.Getenv("HTTP_PROXY"),
		"--env", "HTTPS_PROXY=" + os.Getenv("HTTPS_PROXY"),
		"--env-file", filepath.Join(localDirs.configDir, ResticEnvs),
		imgRestic.ToContainerImage(),
		command,
	}

	args = append(args, extraArgs...)
	out, err := exec.Command("docker", args...).CombinedOutput()
	log.Infoln("Output:", string(out))
	return err
}
