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
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog/v2"
)

type REST struct {
	kubeClient kubernetes.Interface
	config     *rest.Config
}

func newREST(cfg *rest.Config) *REST {
	return &REST{
		kubeClient: kubernetes.NewForConfigOrDie(cfg),
		config:     cfg,
	}
}

func NewCmdUnlockRepository(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "unlock",
		Short:             `Unlock Restic Repository`,
		Long:              `Unlock Restic Repository`,
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
			namespace, _, err = clientGetter.ToRawKubeConfigLoader().Namespace()
			if err != nil {
				return err
			}

			client, err := cs.NewForConfig(cfg)
			if err != nil {
				return err
			}
			// get source repository
			repo, err := client.StashV1alpha1().Repositories(namespace).Get(context.TODO(), repositoryName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			r := newREST(cfg)

			if repo.Spec.Backend.Local != nil {
				return r.unlockLocalRepository(repo)
			}

			return r.unlockRepository(repo)
		},
	}

	return cmd
}

func (r *REST) unlockLocalRepository(repo *v1alpha1.Repository) error {
	if _, err := r.execOnBackendMountingPod("unlock", repo); err != nil {
		return err
	}
	klog.Infof("Repository %s/%s has been unlocked successfully", repo.Namespace, repo.Name)
	return nil
}

const ExecStash = "/stash-enterprise"

func (r *REST) execOnBackendMountingPod(cmd string, repo *v1alpha1.Repository) ([]byte, error) {
	// get the pod that mount this repository as volume
	pod, err := r.getBackendMountingPod(repo)
	if err != nil {
		return nil, err
	}

	command := []string{ExecStash, cmd}
	command = append(command, "--repo-name="+repo.Name)
	command = append(command, "--repo-namespace="+repo.Namespace)

	response, err := r.execCommandOnPod(pod, command)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (r *REST) getBackendMountingPod(repo *v1alpha1.Repository) (*core.Pod, error) {
	vol, mnt := repo.Spec.Backend.Local.ToVolumeAndMount(repo.Name)
	if repo.LocalNetworkVolume() {
		mnt.MountPath = filepath.Join(mnt.MountPath, repo.LocalNetworkVolumePath())
	}
	// list all the pods
	podList, err := r.kubeClient.CoreV1().Pods(repo.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// return the pod that has the vol and mnt
	for i := range podList.Items {
		if hasVolume(podList.Items[i].Spec.Volumes, vol) {
			for _, c := range podList.Items[i].Spec.Containers {
				if hasVolumeMount(c.VolumeMounts, mnt) {
					return &podList.Items[i], nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no backend mounting pod found for Repository %v", repo.Name)
}

func (r *REST) execCommandOnPod(pod *core.Pod, command []string) ([]byte, error) {
	var (
		execOut bytes.Buffer
		execErr bytes.Buffer
	)

	klog.Infof("Executing command %v on pod %v", command, pod.Name)

	req := r.kubeClient.CoreV1().RESTClient().Post().
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

	executor, err := remotecommand.NewSPDYExecutor(r.config, "POST", req.URL())
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

func hasVolume(volumes []core.Volume, vol core.Volume) bool {
	for i := range volumes {
		if volumes[i].Name == vol.Name {
			return true
		}
	}
	return false
}

func hasVolumeMount(mounts []core.VolumeMount, mnt core.VolumeMount) bool {
	for i := range mounts {
		if mounts[i].Name == mnt.Name && mounts[i].MountPath == mnt.MountPath {
			return true
		}
	}
	return false
}

func (r *REST) unlockRepository(repo *v1alpha1.Repository) error {
	// get source repository secret
	secret, err := r.kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), repo.Spec.Backend.StorageSecretName, metav1.GetOptions{})
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
	setupOpt, err := util.SetupOptionsForRepository(*repo, extraOpt)
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

	extraAgrs := []string{
		"--no-cache",
	}

	// For TLS secured Minio/REST server, specify cert path
	if resticWrapper.GetCaPath() != "" {
		extraAgrs = append(extraAgrs, "--cacert", resticWrapper.GetCaPath())
	}

	// run unlock inside docker
	if err = runCmdViaDocker(*localDirs, "unlock", extraAgrs); err != nil {
		return err
	}
	klog.Infof("Repository %s/%s has been unlocked successfully", repo.Namespace, repo.Name)
	return nil
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
		"-v", ScratchDir + ":" + ScratchDir,
		"--env", "HTTP_PROXY=" + os.Getenv("HTTP_PROXY"),
		"--env", "HTTPS_PROXY=" + os.Getenv("HTTPS_PROXY"),
		"--env-file", filepath.Join(localDirs.configDir, ResticEnvs),
		imgRestic.ToContainerImage(),
		command,
	}

	args = append(args, extraArgs...)
	out, err := exec.Command("docker", args...).CombinedOutput()
	klog.Infoln("Output:", string(out))
	return err
}
