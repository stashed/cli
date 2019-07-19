package pkg

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/reference"
	batch_util "kmodules.xyz/client-go/batch/v1"
	v1 "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/tools/cli"
	"stash.appscode.dev/cli/pkg/docker"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	stash_scheme "stash.appscode.dev/stash/client/clientset/versioned/scheme"
	"stash.appscode.dev/stash/pkg/util"
)

const (
	unlockJobPrefix       = "unlock-local-repo-"
	unlockJobSecretDir    = "/etc/secret"
	unlockJobSecretVolume = "secret-volume"
)

func NewCmdUnlockRepository(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	var (
		localDirs = &cliLocalDirectories{}
	)
	var cmd = &cobra.Command{
		Use:               "unlock",
		Short:             `Unlock Restic Repository`,
		Long:              `Unlock Restic Repository`,
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

			// get source repository
			repository, err := client.StashV1alpha1().Repositories(namespace).Get(repositoryName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			// unlock local backend
			if repository.Spec.Backend.Local != nil {
				if err = unlockLocalRepo(kc, repository); err != nil {
					return fmt.Errorf("can't unlock repository for local backend, reason: %s", err)
				}
				return nil
			}
			// get source repository secret
			secret, err := kc.CoreV1().Secrets(namespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// configure restic wrapper
			extraOpt := util.ExtraOptions{
				SecretDir:   docker.SecretDir,
				EnableCache: false,
				ScratchDir:  docker.ScratchDir,
			}
			setupOpt, err := util.SetupOptionsForRepository(*repository, extraOpt)
			if err != nil {
				return fmt.Errorf("setup option for repository failed")
			}

			// write secret and config in a temp dir
			// cleanup whole tempDir dir at the end
			tempDir, err := ioutil.TempDir("", "stash-cli")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tempDir)

			// prepare local dirs
			if err = localDirs.prepareSecretDir(tempDir, secret); err != nil {
				return err
			}
			if err = localDirs.prepareConfigDir(tempDir, &setupOpt, nil); err != nil {
				return err
			}

			// run unlock inside docker
			if err = runUnlockViaDocker(*localDirs); err != nil {
				return err
			}
			log.Infof("Repository %s/%s unlocked", namespace, repositoryName)
			return nil
		},
	}

	cmd.Flags().StringVar(&imgRestic.Registry, "docker-registry", imgRestic.Registry, "Docker image registry")
	cmd.Flags().StringVar(&imgRestic.Tag, "image-tag", imgRestic.Tag, "Stash image tag")

	return cmd
}

func runUnlockViaDocker(localDirs cliLocalDirectories) error {
	// get current user
	currentUser, err := user.Current()
	if err != nil {
		return err
	}
	args := []string{
		"run",
		"--rm",
		"-u", currentUser.Uid,
		"-v", localDirs.configDir + ":" + docker.ConfigDir,
		"-v", localDirs.secretDir + ":" + docker.SecretDir,
		imgRestic.ToContainerImage(),
		"docker",
		"unlock-repository",
	}
	log.Infoln("Running docker with args:", args)
	out, err := exec.Command("docker", args...).CombinedOutput()
	log.Infoln("Output:", string(out))
	return err
}

func unlockLocalRepo(kc kubernetes.Interface, repo *v1alpha1.Repository) error {
	_, path, err := util.GetBucketAndPrefix(&repo.Spec.Backend)
	if err != nil {
		return err
	}

	// create a job and mount secret
	job := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: unlockJobPrefix,
			Namespace:    repo.Namespace,
		},
		Spec: batch.JobSpec{
			BackoffLimit: types.Int32P(1),
			Template: core.PodTemplateSpec{
				Spec: core.PodSpec{
					RestartPolicy: core.RestartPolicyNever,
					Containers: []core.Container{
						{
							Name:  util.StashContainer,
							Image: imgRestic.ToContainerImage(),
							Args: append([]string{
								"cli",
								"unlock-local-repository",
								"--path=" + path,
								"--secret-dir=" + unlockJobSecretDir,
							}, cli.LoggerOptions.ToFlags()...),
							VolumeMounts: []core.VolumeMount{
								{
									Name:      unlockJobSecretVolume,
									MountPath: unlockJobSecretDir,
								},
							},
							ImagePullPolicy: core.PullAlways,
						},
					},
					Volumes: []core.Volume{
						{
							Name: unlockJobSecretVolume,
							VolumeSource: core.VolumeSource{
								Secret: &core.SecretVolumeSource{
									SecretName: repo.Spec.Backend.StorageSecretName,
								},
							},
						},
					},
				},
			},
		},
	}

	// attach local backend
	job.Spec.Template.Spec = util.AttachLocalBackend(job.Spec.Template.Spec, *repo.Spec.Backend.Local)

	// set repository as owner
	ref, err := reference.GetReference(stash_scheme.Scheme, repo)
	if err != nil {
		return err
	}
	v1.EnsureOwnerReference(&job.ObjectMeta, ref)

	job, err = kc.BatchV1().Jobs(repo.Namespace).Create(job)
	if err != nil {
		return err
	}
	log.Infof("Unlock Job %s/%s created, waiting for completion...", job.Namespace, job.Name)

	// cleanup unlock job // TODO: keep or remove ?
	/*defer func() {
		err := kc.BatchV1().Jobs(repo.Namespace).Delete(job.Name, &metav1.DeleteOptions{})
		if err != nil {
			log.Errorln(err)
		}
	}()*/

	// wait for job to complete
	if err = batch_util.WaitUntilJobCompletion(kc, job.ObjectMeta); err != nil {
		return err
	}

	// check job status
	job, err = kc.BatchV1().Jobs(repo.Namespace).Get(job.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if job.Status.Succeeded > 0 {
		log.Infof("Unlock Job %s/%s succeeded", job.Namespace, job.Name)
		return nil
	}
	return fmt.Errorf("unlock Job %s/%s failed", job.Namespace, job.Name)
}
