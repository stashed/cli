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
	"strings"

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
	"k8s.io/klog/v2"
	aggcs "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

type downloadOptions struct {
	kubeClient *kubernetes.Clientset
	config     *rest.Config
	repo       *v1alpha1.Repository
	localDirs  *cliLocalDirectories

	SourceHost   string
	RestorePaths []string
	Snapshots    []string
	Destination  string
}

func NewCmdDownloadRepository(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	opt := downloadOptions{
		SourceHost:  restic.DefaultHost,
		Destination: DestinationDir,
		localDirs:   &cliLocalDirectories{},
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

			if err = opt.localDirs.prepareDownloadDir(); err != nil {
				return err
			}

			if opt.repo.Spec.Backend.Local != nil {
				// get the pod that mount this repository as volume
				pod, err := getBackendMountingPod(opt.kubeClient, opt.repo)
				if err != nil {
					return err
				}
				return opt.downloadSnapshotsFromPod(pod, opt.Snapshots)
			}

			aggrClient, err = aggcs.NewForConfig(opt.config)
			if err != nil {
				return err
			}

			operatorPod, err := GetOperatorPod(aggrClient, opt.kubeClient)
			if err != nil {
				return err
			}
			yes, err := opt.isPodServiceAccountAnnotatedForIdentity(operatorPod)
			if err != nil {
				return err
			}
			if yes {
				return opt.downloadSnapshotsFromPod(operatorPod, opt.Snapshots)
			}

			return opt.downloadSnapshots()
		},
	}

	cmd.Flags().StringVar(&opt.localDirs.downloadDir, "destination", opt.localDirs.downloadDir, "Destination path where snapshot will be restored.")

	cmd.Flags().StringVar(&opt.SourceHost, "host", opt.SourceHost, "Name of the source host machine")
	cmd.Flags().StringSliceVar(&opt.RestorePaths, "paths", opt.RestorePaths, "List of directories to be restored")
	cmd.Flags().StringSliceVar(&opt.Snapshots, "snapshots", opt.Snapshots, "List of snapshots to be restored")

	cmd.Flags().StringVar(&imgRestic.Registry, "docker-registry", imgRestic.Registry, "Docker image registry for restic cli")
	cmd.Flags().StringVar(&imgRestic.Tag, "image-tag", imgRestic.Tag, "Restic docker image tag")

	return cmd
}

func (opt *downloadOptions) isPodServiceAccountAnnotatedForIdentity(pod *core.Pod) (bool, error) {
	serviceAccount, err := opt.kubeClient.CoreV1().ServiceAccounts(pod.Namespace).Get(context.TODO(), pod.Spec.ServiceAccountName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	googleAnnotation := "iam.gke.io/gcp-service-account"
	awsAnnotation := "eks.amazonaws.com/role-arn"

	if _, exists := serviceAccount.Annotations[googleAnnotation]; exists {
		return true, nil
	} else if _, exists := serviceAccount.Annotations[awsAnnotation]; exists {
		return true, nil
	}

	return false, nil
}

func (opt *downloadOptions) downloadSnapshots() error {
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

	opt.localDirs.configDir = filepath.Join(ScratchDir, configDirName)
	// dump restic's environments into `restic-env` file.
	// we will pass this env file to restic docker container.
	err = resticWrapper.DumpEnv(opt.localDirs.configDir, ResticEnvs)
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
	if err = runRestoreViaDocker(*opt.localDirs, extraAgrs, opt.Snapshots); err != nil {
		return err
	}
	klog.Infof("Snapshots: %v of Repository %s/%s restored in path %s", opt.Snapshots, namespace, opt.repo.Name, opt.localDirs.downloadDir)
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
		klog.Infoln("Output:", string(out))
		if err != nil {
			return err
		}
	}
	return nil
}

func (opt *downloadOptions) downloadSnapshotsFromPod(pod *core.Pod, snapshots []string) error {
	if err := opt.executeDownloadCmdInPod(pod, snapshots); err != nil {
		return err
	}
	if err := opt.copyDownloadedDataToDestination(pod); err != nil {
		return err
	}

	if err := opt.clearDataFromPod(pod); err != nil {
		return err
	}

	klog.Infof("Snapshots: %v of Repository %s/%s restored in path %s", snapshots, namespace, opt.repo.Name, opt.localDirs.downloadDir)
	return nil
}

func (opt *downloadOptions) executeDownloadCmdInPod(pod *core.Pod, snapshots []string) error {
	command := []string{"/stash-enterprise", "download"}
	command = append(command, "--repo-name", opt.repo.Name, "--repo-namespace", opt.repo.Namespace)
	command = append(command, "--snapshots", strings.Join(snapshots, ","))
	command = append(command, "--destination", opt.getPodDirForSnapshots())

	out, err := execCommandOnPod(opt.kubeClient, opt.config, pod, command)
	if string(out) != "" {
		klog.Infoln("Output:", string(out))
	}
	return err
}

func (opt *downloadOptions) copyDownloadedDataToDestination(pod *core.Pod) error {
	_, err := exec.Command(cmdKubectl, "cp", "--namespace", pod.Namespace, fmt.Sprintf("%s/%s:%s", pod.Namespace, pod.Name, opt.getPodDirForSnapshots()), opt.localDirs.downloadDir).CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

func (opt *downloadOptions) clearDataFromPod(pod *core.Pod) error {
	cmd := []string{"rm", "-rf", opt.getPodDirForSnapshots()}
	out, err := execCommandOnPod(opt.kubeClient, opt.config, pod, cmd)
	if string(out) != "" {
		klog.Infoln("Output:", string(out))
	}
	return err
}

func (opt *downloadOptions) getPodDirForSnapshots() string {
	return filepath.Join(apis.ScratchDirMountPath, apis.SnapshotDownloadDir)
}
