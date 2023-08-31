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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"stash.appscode.dev/apimachinery/apis"
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
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/scheme"
)

type downloadOptions struct {
	kubeClient kubernetes.Interface
	config     *rest.Config
	repo       *v1alpha1.Repository
}

func newDownloadOptions(cfg *rest.Config, repo *v1alpha1.Repository) *downloadOptions {
	return &downloadOptions{
		kubeClient: kubernetes.NewForConfigOrDie(cfg),
		config:     cfg,
		repo:       repo,
	}
}

var localDirs = &cliLocalDirectories{}

func NewCmdDownloadRepository(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	restoreOpt := restic.RestoreOptions{
		SourceHost:  restic.DefaultHost,
		Destination: apis.DestinationDir,
	}
	cmd := &cobra.Command{
		Use:               "download",
		Short:             `Download snapshots`,
		Long:              `Download contents of snapshots from Repository`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("repository name not found")
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

			client, err := cs.NewForConfig(cfg)
			if err != nil {
				return err
			}

			repository, err := client.StashV1alpha1().Repositories(namespace).Get(context.TODO(), repositoryName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if err = localDirs.prepareDownloadDir(); err != nil {
				return err
			}

			opt := newDownloadOptions(cfg, repository)

			if repository.Spec.Backend.Local != nil {
				return opt.downloadSnapshotsFromLocalRepo(restoreOpt.Snapshots)
			}

			return opt.downloadSnapshots(&restoreOpt)
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

func (opt *downloadOptions) downloadSnapshots(restoreOpt *restic.RestoreOptions) error {
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
	klog.Infof("Snapshots: %v of Repository %s/%s restored in path %s", restoreOpt.Snapshots, namespace, opt.repo.Name, localDirs.downloadDir)
	return nil
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
		"-v", localDirs.downloadDir + ":" + apis.DestinationDir,
		"--env", "HTTP_PROXY=" + os.Getenv("HTTP_PROXY"),
		"--env", "HTTPS_PROXY=" + os.Getenv("HTTPS_PROXY"),
		"--env-file", filepath.Join(localDirs.configDir, ResticEnvs),
		imgRestic.ToContainerImage(),
	}

	restoreArgs = append(restoreArgs, extraArgs...)
	restoreArgs = append(restoreArgs, "restore")
	for _, snapshot := range snapshots {
		args := append(restoreArgs, snapshot, "--target", filepath.Join(apis.DestinationDir, snapshot))
		klog.Infoln("Running docker with args:", args)
		out, err := exec.Command("docker", args...).CombinedOutput()
		if err != nil {
			return err
		}
		klog.Infoln("Output:", string(out))
	}
	return nil
}

func (opt *downloadOptions) downloadSnapshotsFromLocalRepo(snapshots []string) error {
	// get the pod that mount this repository as volume
	pod, err := getBackendMountingPod(opt.kubeClient, opt.repo)
	if err != nil {
		return err
	}

	if err := opt.downloadSnapshotsInMountingPod(pod, snapshots); err != nil {
		return err
	}
	if err := opt.copyDownloadedDataToDestination(pod); err != nil {
		return err
	}

	if err := opt.clearDataFromMountingPod(pod); err != nil {
		return err
	}

	klog.Infof("Snapshots have been downloaded successfully", opt.repo.Namespace, opt.repo.Name)
	return nil
}

func (opt *downloadOptions) execCommandOnPod(pod *core.Pod, command []string) ([]byte, error) {
	var (
		execOut bytes.Buffer
		execErr bytes.Buffer
	)

	klog.Infof("Executing command %v on pod %v", command, pod.Name)

	req := opt.kubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		Timeout(5 * time.Minute)
	req.VersionedParams(&core.PodExecOptions{
		Container: apis.StashContainer,
		Command:   command,
		Stdout:    true,
		Stderr:    true,
	}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(opt.config, "POST", req.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to init executor: %v", err)
	}

	err = executor.Stream(remotecommand.StreamOptions{
		Stdout: &execOut,
		Stderr: &execErr,
		Tty:    true,
	})

	if err != nil {
		return nil, fmt.Errorf("could not execute: %v, reason: %s", err, execErr.String())
	}

	return execOut.Bytes(), nil
}

func (opt *downloadOptions) downloadSnapshotsInMountingPod(pod *core.Pod, snapshots []string) error {
	command := []string{"/stash-enterprise", "download"}
	command = append(command, "--repo-name", opt.repo.Name)
	command = append(command, "--repo-namespace", opt.repo.Namespace)
	command = append(command, "--snapshots", strings.Join(snapshots, ","))

	_, err := opt.execCommandOnPod(pod, command)
	return err
}

func (opt *downloadOptions) copyDownloadedDataToDestination(pod *core.Pod) error {
	_, err := exec.Command("kubectl", "cp", "--namespace", pod.Namespace, fmt.Sprintf("%s:%s", pod.Name, apis.DestinationDir), localDirs.downloadDir).CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

func (opt *downloadOptions) clearDataFromMountingPod(pod *core.Pod) error {
	cmd := []string{"rm", "-rf", apis.DestinationDir}
	_, err := opt.execCommandOnPod(pod, cmd)
	return err
}
