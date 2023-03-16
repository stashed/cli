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

	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func NewCmdDownloadRepository(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	var (
		localDirs  = &cliLocalDirectories{}
		restoreOpt = restic.RestoreOptions{
			SourceHost:  restic.DefaultHost,
			Destination: DestinationDir,
		}
	)
	cmd := &cobra.Command{
		Use:               "download",
		Short:             `Download snapshots`,
		Long:              `Download contents of snapshots from Repository`,
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

			repository, err := client.StashV1alpha1().Repositories(namespace).Get(context.TODO(), repositoryName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if repository.Spec.Backend.Local != nil {
				return fmt.Errorf("can't restore from repository with local backend")
			}

			// get source repository secret
			secret, err := kc.CoreV1().Secrets(namespace).Get(context.TODO(), repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if err = localDirs.prepareDownloadDir(); err != nil {
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
			setupOpt, err := util.SetupOptionsForRepository(*repository, extraOpt)
			if err != nil {
				return fmt.Errorf("setup option for repository failed")
			}

			resticWrapper, err := restic.NewResticWrapper(setupOpt)
			if err != nil {
				return err
			}

			localDirs.configDir = filepath.Join(ScratchDir, configDirName)
			// dump restic's environments into `restic-env` file.
			// we will pass this env file to restic docker container.
			err = resticWrapper.DumpEnv(localDirs.configDir, ResticEnvs)
			if err != nil {
				return err
			}

			extraAgrs := []string{
				"--no-cache",
			}

			// For TLS secured Minio/REST server, specify cert path
			if resticWrapper.GetCaPath() != "" {
				extraAgrs = append(extraAgrs, "--cacert", resticWrapper.GetCaPath())
			}

			// run restore inside docker
			if err = runRestoreViaDocker(*localDirs, extraAgrs, restoreOpt.Snapshots); err != nil {
				return err
			}
			klog.Infof("Snapshots: %v of Repository %s/%s restored in path %s", restoreOpt.Snapshots, namespace, repositoryName, localDirs.downloadDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&localDirs.downloadDir, "destination", localDirs.downloadDir, "Destination path where snapshot will be restored.")

	cmd.Flags().StringVar(&restoreOpt.SourceHost, "host", restoreOpt.SourceHost, "Name of the source host machine")
	cmd.Flags().StringSliceVar(&restoreOpt.RestorePaths, "paths", restoreOpt.RestorePaths, "List of directories to be restored")
	cmd.Flags().StringSliceVar(&restoreOpt.Snapshots, "snapshots", restoreOpt.Snapshots, "List of snapshots to be restored")

	cmd.Flags().StringVar(&imgRestic.Registry, "docker-registry", imgRestic.Registry, "Docker image registry for restic cli")
	cmd.Flags().StringVar(&imgRestic.Tag, "image-tag", imgRestic.Tag, "Restic docker image tag")

	return cmd
}

func runRestoreViaDocker(localDirs cliLocalDirectories, extraArgs []string, snapshots []string) error {
	// get current user
	currentUser, err := user.Current()
	if err != nil {
		return err
	}
	restoreArgs := []string{
		"run",
		"--rm",
		"-u", currentUser.Uid,
		"-v", ScratchDir + ":" + ScratchDir,
		"-v", localDirs.downloadDir + ":" + DestinationDir,
		"--env", "HTTP_PROXY=" + os.Getenv("HTTP_PROXY"),
		"--env", "HTTPS_PROXY=" + os.Getenv("HTTPS_PROXY"),
		"--env-file", filepath.Join(localDirs.configDir, ResticEnvs),
		imgRestic.ToContainerImage(),
	}

	restoreArgs = append(restoreArgs, extraArgs...)
	restoreArgs = append(restoreArgs, "restore")
	for _, snapshot := range snapshots {
		args := append(restoreArgs, snapshot, "--target", filepath.Join(DestinationDir, snapshot))
		klog.Infoln("Running docker with args:", args)
		out, err := exec.Command("docker", args...).CombinedOutput()
		if err != nil {
			return err
		}
		klog.Infoln("Output:", string(out))
	}
	return nil
}
