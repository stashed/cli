/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pkg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gomodules.xyz/flags"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type keyOptions struct {
	config    *rest.Config
	repo      *v1alpha1.Repository
	localDirs cliLocalDirectories
	restic.KeyOptions
}

func NewCmdAddKey(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	opt := keyOptions{}
	cmd := &cobra.Command{
		Use:               "add",
		Short:             `Add a new key (password) to a restic repository`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "new-password-file")
			var err error
			opt.File, err = filepath.Abs(opt.File)
			if err != nil {
				return fmt.Errorf("failed to find the absolute path for password file: %w", err)
			}

			_, err = os.Stat(opt.File)
			if os.IsNotExist(err) {
				return fmt.Errorf("%s does not exist", opt.File)
			}

			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("repository name not found")
			}
			repositoryName := args[0]

			opt.config, err = clientGetter.ToRESTConfig()
			if err != nil {
				return errors.Wrap(err, "failed to read kubeconfig")
			}

			// get source repository
			opt.repo, err = stashClient.StashV1alpha1().Repositories(namespace).Get(context.TODO(), repositoryName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if opt.repo.Spec.Backend.Local != nil {
				return opt.addResticKeyForLocalRepo()
			}

			return opt.addResticKey()
		},
	}

	cmd.Flags().StringVar(&opt.Host, "host", opt.Host, "host for the new key")
	cmd.Flags().StringVar(&opt.User, "user", opt.User, "username for the new key")
	cmd.Flags().StringVar(&opt.File, "new-password-file", opt.File, "file from which to read the new password")

	return cmd
}

func (opt *keyOptions) addResticKeyForLocalRepo() error {
	// get the pod that mount this repository as volume
	pod, err := getBackendMountingPod(kubeClient, opt.repo)
	if err != nil {
		return err
	}

	if err := opt.copyPasswordFileToPod(pod); err != nil {
		return fmt.Errorf("failed to copy password file from local directory to pod: %w", err)
	}

	command := []string{"/stash-enterprise", "add-key"}
	command = append(command, "--repo-name="+opt.repo.Name, "--repo-namespace="+opt.repo.Namespace)
	command = append(command, "--new-password-file="+getPodDirForPasswordFile())

	if opt.User != "" {
		command = append(command, "--user="+opt.User)
	}
	if opt.Host != "" {
		command = append(command, "--host="+opt.Host)
	}

	_, err = execCommandOnPod(kubeClient, opt.config, pod, command)
	if err != nil {
		return err
	}

	if err := opt.removePasswordFileFromPod(pod); err != nil {
		return fmt.Errorf("failed to remove password file from pod: %w", err)
	}

	klog.Infof("Restic key has been added successfully for repository %s/%s", opt.repo.Namespace, opt.repo.Name)
	return nil
}

func (opt *keyOptions) removePasswordFileFromPod(pod *core.Pod) error {
	cmd := []string{"rm", "-rf", getPodDirForPasswordFile()}
	out, err := execCommandOnPod(kubeClient, opt.config, pod, cmd)
	if string(out) != "" {
		klog.Infoln("Output:", string(out))
	}
	return err
}

func (opt *keyOptions) copyPasswordFileToPod(pod *core.Pod) error {
	_, err := exec.Command(cmdKubectl, "cp", opt.File, fmt.Sprintf("%s/%s:%s", pod.Namespace, pod.Name, getPodDirForPasswordFile())).CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

func getPodDirForPasswordFile() string {
	return filepath.Join(apis.ScratchDirMountPath, passwordFile)
}

func (opt *keyOptions) addResticKey() error {
	// get source repository secret
	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), opt.repo.Spec.Backend.StorageSecretName, metav1.GetOptions{})
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
	// init restic wrapper
	resticWrapper, err := restic.NewResticWrapper(setupOpt)
	if err != nil {
		return err
	}

	opt.localDirs = cliLocalDirectories{
		configDir: filepath.Join(ScratchDir, configDirName),
	}

	// dump restic's environments into `restic-env` file.
	// we will pass this env file to restic docker container.

	err = resticWrapper.DumpEnv(opt.localDirs.configDir, ResticEnvs)
	if err != nil {
		return err
	}

	args := []string{
		"key",
		"add",
		"--no-cache",
	}

	// For TLS secured Minio/REST server, specify cert path
	if resticWrapper.GetCaPath() != "" {
		args = append(args, "--cacert", resticWrapper.GetCaPath())
	}

	if err = manageKeyViaDocker(opt, args); err != nil {
		return err
	}
	klog.Infof("Restic key has been added successfully for repository %s/%s", opt.repo.Namespace, opt.repo.Name)
	return nil
}

func manageKeyViaDocker(opt *keyOptions, args []string) error {
	// get current user
	currentUser, err := user.Current()
	if err != nil {
		return err
	}

	keyArgs := []string{
		"run",
		"--rm",
		"-u", currentUser.Uid,
		"-v", ScratchDir + ":" + ScratchDir,
		"--env", "HTTP_PROXY=" + os.Getenv("HTTP_PROXY"),
		"--env", "HTTPS_PROXY=" + os.Getenv("HTTPS_PROXY"),
		"--env-file", filepath.Join(opt.localDirs.configDir, ResticEnvs),
	}

	if opt.File != "" {
		keyArgs = append(keyArgs, "-v", opt.File+":"+opt.File)
	}

	keyArgs = append(keyArgs, imgRestic.ToContainerImage())
	keyArgs = append(keyArgs, args...)

	if opt.File != "" {
		keyArgs = append(keyArgs, "--new-password-file", opt.File)
	}

	if opt.User != "" {
		keyArgs = append(keyArgs, "--user", opt.User)
	}

	if opt.Host != "" {
		keyArgs = append(keyArgs, "--host", opt.Host)
	}

	klog.Infoln("Running docker with args:", keyArgs)

	out, err := exec.Command("docker", keyArgs...).CombinedOutput()
	if err != nil {
		return err
	}
	klog.Infoln(fmt.Sprintf("\n%s", string(out)))
	return nil
}
