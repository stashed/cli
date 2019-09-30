package pkg

import (
	"fmt"
	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubectl/util/templates"
	"kmodules.xyz/objectstore-api/api/v1"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
	"stash.appscode.dev/stash/pkg/restic"
)

var (
	createRepositoryExample = templates.Examples(`
		# Create a new repository
		stash create repository --namespace=<namespace> <repository-name> [Flag]
        stash create repository gcs-repo --namespace=demo --secret=gcs-secret --bucket=appscode-qa --prefix=/source/data --provider=gcs`)
)

type repositoryOption struct {
	provider       string
	bucket         string
	endpoint       string
	maxConnections int
	secret         string
	prefix         string
}

func NewCmdCreateRepository() *cobra.Command {
	var repoOpt = repositoryOption{}
	var cmd = &cobra.Command{
		Use:               "repository",
		Short:             `Create a new repository`,
		Long:              "Create a new Repository using Backend Credential",
		Example:           createRepositoryExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("Repository name is not provided ")
			}

			repositoryName := args[0]

			repository := newRepository(repoOpt, repositoryName, namespace)

			repository, err := createRepository(repository, repository.ObjectMeta)
			if err != nil {
				return err
			}
			log.Infof("Repository %s/%s has been created successfully.", repository.Namespace, repository.Name)
			return err

		},
	}
	cmd.Flags().StringVar(&repoOpt.provider, "provider", repoOpt.provider, "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&repoOpt.bucket, "bucket", repoOpt.bucket, "Name of the cloud bucket/container")
	cmd.Flags().StringVar(&repoOpt.endpoint, "endpoint", repoOpt.endpoint, "Endpoint for s3/s3 compatible backend")
	cmd.Flags().IntVar(&repoOpt.maxConnections, "max-connections", repoOpt.maxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")
	cmd.Flags().StringVar(&repoOpt.secret, "secret", repoOpt.secret, "Name of the Storage Secret")
	cmd.Flags().StringVar(&repoOpt.prefix, "prefix", repoOpt.prefix, "Prefix denotes the directory inside the backend")

	return cmd
}

func newRepository(opt repositoryOption, name string, namespace string) *v1alpha1.Repository {
	repository := &v1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.RepositorySpec{
			Backend: opt.getBackendInfo(),
		},
	}
	return repository
}

// CreateOrPatch New Secret
func createRepository(repository *v1alpha1.Repository, meta metav1.ObjectMeta) (*v1alpha1.Repository, error) {
	repository, _, err := util.CreateOrPatchRepository(stashClient.StashV1alpha1(), meta, func(in *v1alpha1.Repository) *v1alpha1.Repository {
		in.Spec = repository.Spec
		return in
	},
	)
	return repository, err
}

func (opt repositoryOption) getBackendInfo() v1.Backend {
	var backend v1.Backend
	switch opt.provider {
	case restic.ProviderGCS:
		backend = v1.Backend{
			GCS: &v1.GCSSpec{
				Bucket:         opt.bucket,
				Prefix:         opt.prefix,
				MaxConnections: opt.maxConnections,
			},
		}
	case restic.ProviderAzure:
		backend = v1.Backend{
			Azure: &v1.AzureSpec{
				Container:      opt.bucket,
				Prefix:         opt.prefix,
				MaxConnections: opt.maxConnections,
			},
		}
	case restic.ProviderS3:
		backend = v1.Backend{
			S3: &v1.S3Spec{
				Bucket:   opt.bucket,
				Prefix:   opt.prefix,
				Endpoint: opt.endpoint,
			},
		}
	case restic.ProviderB2:
		backend = v1.Backend{
			B2: &v1.B2Spec{
				Bucket:         opt.bucket,
				Prefix:         opt.prefix,
				MaxConnections: opt.maxConnections,
			},
		}
	case restic.ProviderSwift:
		backend = v1.Backend{
			Swift: &v1.SwiftSpec{
				Container: opt.bucket,
				Prefix:    opt.prefix,
			},
		}
	case restic.ProviderRest:
		backend = v1.Backend{
			Rest: &v1.RestServerSpec{
				URL: opt.endpoint,
			},
		}
	}
	backend.StorageSecretName = opt.secret
	return backend
}