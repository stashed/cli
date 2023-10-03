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
	"path/filepath"

	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gomodules.xyz/flags"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
)

func NewCmdUpdateKey(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	opt := keyOptions{}
	cmd := &cobra.Command{
		Use:               "update",
		Short:             `Update current key (password) of a restic repository`,
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
				return opt.updateResticKeyForLocalRepo()
			}

			return opt.updateResticKey()
		},
	}

	cmd.Flags().StringVar(&opt.File, "new-password-file", opt.File, "file from which to read the new password")

	return cmd
}

func (opt *keyOptions) updateResticKeyForLocalRepo() error {
	// get the pod that mount this repository as volume
	pod, err := getBackendMountingPod(kubeClient, opt.repo)
	if err != nil {
		return err
	}

	if err := opt.copyPasswordFileToPod(pod); err != nil {
		return fmt.Errorf("failed to copy password file from local directory to pod: %w", err)
	}

	command := []string{"/stash-enterprise", "update-key"}
	command = append(command, "--repo-name="+opt.repo.Name, "--repo-namespace="+opt.repo.Namespace)
	command = append(command, "--new-password-file="+getPodDirForPasswordFile())

	_, err = execCommandOnPod(kubeClient, opt.config, pod, command)
	if err != nil {
		return err
	}

	if err := opt.removePasswordFileFromPod(pod); err != nil {
		return fmt.Errorf("failed to remove password file from pod: %w", err)
	}

	klog.Infof("Restic key has been updated successfully for repository %s/%s", opt.repo.Namespace, opt.repo.Name)
	return nil
}

func (opt *keyOptions) updateResticKey() error {
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
		"passwd",
		"--no-cache",
	}

	// For TLS secured Minio/REST server, specify cert path
	if resticWrapper.GetCaPath() != "" {
		args = append(args, "--cacert", resticWrapper.GetCaPath())
	}

	if err = manageKeyViaDocker(opt, args); err != nil {
		return err
	}
	klog.Infof("Restic key has been updated successfully for repository %s/%s", opt.repo.Namespace, opt.repo.Name)
	return nil
}
