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
	"stash.appscode.dev/stash/pkg/registry/snapshot"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/util/templates"
)

var deleteExample = templates.Examples(`
		# Delete Snapshot
		stash delete snapshot gcs-repo-c063d146 -n demo`)

func NewCmdDeleteSnapshot(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	localDirs := &cliLocalDirectories{}

	cmd := &cobra.Command{
		Use:               "snapshot",
		Short:             `Delete a snapshot from repository backend`,
		Long:              `Delete a snapshot from repository backend`,
		DisableAutoGenTag: true,
		Example:           deleteExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("snapshot name not provided")
			}
			repoName, snapshotId, err := util.GetRepoNameAndSnapshotID(args[0])
			if err != nil {
				return err
			}

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
			repository, err := client.StashV1alpha1().Repositories(namespace).Get(context.TODO(), repoName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// get source repository secret
			secret, err := kc.CoreV1().Secrets(namespace).Get(context.TODO(), repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			opt := snapshot.Options{
				Repository:  repository,
				Secret:      secret,
				SnapshotIDs: []string{snapshotId},
				InCluster:   false,
			}

			// delete from local backend
			if repository.Spec.Backend.Local != nil {
				r := snapshot.NewREST(cfg)
				return r.ForgetSnapshotsFromBackend(opt)
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

			// run unlock inside docker
			if err = runDeleteSnapshotViaDocker(*localDirs, extraAgrs, snapshotId); err != nil {
				return err
			}
			klog.Infof("Snapshot %s deleted from repository %s/%s", snapshotId, namespace, repoName)
			return nil
		},
	}

	cmd.Flags().StringVar(&imgRestic.Registry, "docker-registry", imgRestic.Registry, "Docker image registry")
	cmd.Flags().StringVar(&imgRestic.Tag, "image-tag", imgRestic.Tag, "Stash image tag")

	return cmd
}

func runDeleteSnapshotViaDocker(localDirs cliLocalDirectories, extraArgs []string, snapshotId string) error {
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
		"forget", snapshotId,
		"--prune",
	}
	args = append(args, extraArgs...)
	klog.Infoln("Running docker with args:", args)
	out, err := exec.Command("docker", args...).CombinedOutput()
	klog.Infoln("Output:", string(out))
	return err
}
