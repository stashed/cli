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

	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type checkOptions struct {
	kubeClient *kubernetes.Clientset
	config     *rest.Config
	repo       *v1alpha1.Repository

	// All restic options for the 'check' command.
	readData       bool
	readDataSubset string
	withCache      bool
}

func NewCmdCheckRepository(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	opt := checkOptions{}
	cmd := &cobra.Command{
		Use:               "check",
		Short:             `Check the repository for errors`,
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

			opt.kubeClient, err = kubernetes.NewForConfig(opt.config)
			if err != nil {
				return err
			}

			stashClient, err = cs.NewForConfig(opt.config)
			if err != nil {
				return err
			}

			// get source repository
			opt.repo, err = stashClient.StashV1alpha1().Repositories(namespace).Get(context.TODO(), repositoryName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			extraArgs := opt.getUserExtraArguments()
			if opt.repo.Spec.Backend.Local != nil {
				return opt.checkLocalRepository(extraArgs)
			}

			return opt.checkRepository(extraArgs)
		},
	}

	cmd.Flags().BoolVar(&opt.readData, "read-data", false, "read all data blobs")
	cmd.Flags().BoolVar(&opt.withCache, "with-cache", false, "use existing cache, only read uncached data from repository")
	cmd.Flags().StringVar(&opt.readDataSubset, "read-data-subset", "", "read a `subset` of data packs, specified as 'n/t' for specific part, or either 'x%' or 'x.y%' or a size in bytes with suffixes k/K, m/M, g/G, t/T for a random subset")
	return cmd
}

func (opt *checkOptions) checkLocalRepository(extraArgs []string) error {
	// get the pod that mount this repository as volume
	pod, err := getBackendMountingPod(opt.kubeClient, opt.repo)
	if err != nil {
		return err
	}

	command := []string{"/stash-enterprise", "check"}
	command = append(command, extraArgs...)
	command = append(command, "--repo-name="+opt.repo.Name, "--repo-namespace="+opt.repo.Namespace)

	out, err := execCommandOnPod(opt.kubeClient, opt.config, pod, command)
	if string(out) != "" {
		klog.Infoln("Output:", string(out))
	}
	if err != nil {
		return err
	}
	klog.Infof("Repository %s/%s has been checked successfully", opt.repo.Namespace, opt.repo.Name)
	return nil
}

func (opt *checkOptions) checkRepository(extraArgs []string) error {
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
	// init restic wrapper
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

	// For TLS secured Minio/REST server, specify cert path
	if resticWrapper.GetCaPath() != "" {
		extraArgs = append(extraArgs, "--cacert", resticWrapper.GetCaPath())
	}

	// run unlock inside docker
	if err = runCmdViaDocker(*localDirs, "check", extraArgs); err != nil {
		return err
	}
	klog.Infof("Repository %s/%s has been checked successfully", opt.repo.Namespace, opt.repo.Name)
	return nil
}

func (opt *checkOptions) getUserExtraArguments() []string {
	var extraArgs []string
	if opt.readData {
		extraArgs = append(extraArgs, "--read-data")
	}
	if opt.readDataSubset != "" {
		extraArgs = append(extraArgs,
			fmt.Sprintf("--read-data-subset=%s", opt.readDataSubset))
	}
	if opt.withCache {
		extraArgs = append(extraArgs, "--with-cache")
	}
	return extraArgs
}
