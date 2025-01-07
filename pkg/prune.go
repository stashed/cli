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
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type pruneOptions struct {
	kubeClient *kubernetes.Clientset
	config     *rest.Config
	repo       *v1alpha1.Repository

	maxUnusedLimit string
	maxRepackSize  string
	dryRun         bool
}

func NewCmdPruneRepository(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	opt := pruneOptions{}

	cmd := &cobra.Command{
		Use:               "prune",
		Short:             `Prune restic repository`,
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

			extraArgs := opt.getUserExtraArguments()
			if opt.repo.Spec.Backend.Local != nil {
				// get the pod that mount this repository as volume
				pod, err := getBackendMountingPod(opt.kubeClient, opt.repo)
				if err != nil {
					return err
				}
				return opt.pruneRepoFromPod(pod, extraArgs)
			}

			return opt.pruneRepo(extraArgs)
		},
	}

	cmd.Flags().StringVar(&opt.maxUnusedLimit, "max-unused-limit", "", "allow unused data up to the specified limit within the repository")
	cmd.Flags().StringVar(&opt.maxRepackSize, "max-repack-size", "", "limit the total size of files to repack")
	cmd.Flags().BoolVar(&opt.dryRun, "dry-run", false, "only show what prune would do")

	cmd.Flags().StringVar(&imgRestic.Registry, "docker-registry", imgRestic.Registry, "Docker image registry for restic cli")
	cmd.Flags().StringVar(&imgRestic.Tag, "image-tag", imgRestic.Tag, "Restic docker image tag")

	return cmd
}

func (opt *pruneOptions) pruneRepoFromPod(pod *core.Pod, extraArgs []string) error {
	if err := opt.executePruneRepoCmdInPod(pod, extraArgs); err != nil {
		return err
	}

	klog.Infof("Repository %s/%s is pruned", namespace, opt.repo.Name)
	return nil
}

func (opt *pruneOptions) executePruneRepoCmdInPod(pod *core.Pod, extraArgs []string) error {
	command := []string{"/stash-enterprise", "prune"}
	command = append(command, extraArgs...)
	command = append(command, "--repo-name", opt.repo.Name, "--repo-namespace", opt.repo.Namespace)

	out, err := execCommandOnPod(opt.kubeClient, opt.config, pod, command)
	if string(out) != "" {
		klog.Infoln("Output:", string(out))
	}
	return err
}

func (opt *pruneOptions) pruneRepo(extraArgs []string) error {
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

	// For TLS secured Minio/REST server, specify cert path
	if resticWrapper.GetCaPath() != "" {
		extraArgs = append(extraArgs, "--cacert", resticWrapper.GetCaPath())
	}

	// run restore inside docker
	if err = runCmdViaDocker(*localDirs, "prune", extraArgs); err != nil {
		return err
	}
	klog.Infof("Repository %s/%s is pruned", namespace, opt.repo.Name)
	return nil
}

func (opt *pruneOptions) getUserExtraArguments() []string {
	var extraArgs []string
	if opt.maxUnusedLimit != "" {
		extraArgs = append(extraArgs,
			fmt.Sprintf("--max-unused=%s", opt.maxUnusedLimit))
	}
	if opt.maxRepackSize != "" {
		extraArgs = append(extraArgs,
			fmt.Sprintf("--max-repack-size=%s", opt.maxRepackSize))
	}
	if opt.dryRun {
		extraArgs = append(extraArgs, "--dry-run")
	}
	return extraArgs
}
