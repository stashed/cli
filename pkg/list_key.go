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

func NewCmdListKey(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	opt := keyOptions{}
	cmd := &cobra.Command{
		Use:               "list",
		Short:             `list restic keys`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "new-password-file")

			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("repository name not found")
			}
			repositoryName := args[0]

			var err error
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
				return opt.listResticKeyForLocalRepo()
			}

			return opt.listResticKey()
		},
	}

	return cmd
}

func (opt *keyOptions) listResticKeyForLocalRepo() error {
	// get the pod that mount this repository as volume
	pod, err := getBackendMountingPod(kubeClient, opt.repo)
	if err != nil {
		return err
	}

	command := []string{"/stash-enterprise", "list-key"}
	command = append(command, "--repo-name="+opt.repo.Name, "--repo-namespace="+opt.repo.Namespace)

	out, err := execCommandOnPod(kubeClient, opt.config, pod, command)
	if err != nil {
		return err
	}

	klog.Infoln(fmt.Sprintf("\n%s", string(out)))

	return nil
}

func (opt *keyOptions) listResticKey() error {
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
		"list",
		"--no-cache",
	}

	// For TLS secured Minio/REST server, specify cert path
	if resticWrapper.GetCaPath() != "" {
		args = append(args, "--cacert", resticWrapper.GetCaPath())
	}

	// run unlock inside docker
	if err = manageKeyViaDocker(opt, args); err != nil {
		return err
	}
	return nil
}
