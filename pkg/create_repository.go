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
	createRepositoryLong = templates.LongDesc(`
		Create a new Repository`)

	createRepositoryExample = templates.Examples(`
		# Create a new repository
		stash create repository --namespace=<namespace> <repository-name> [Flag]
        stash create repository gcs-repo --namespace=demo --secret=gcs-secret --bucket=appscode-qa --prefix=/source/data --provider=gcs`)
)

func NewCmdCreateRepository() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "repository",
		Short:             `Create a repository`,
		Long:              createRepositoryExample,
		Example:           createRepositoryExample,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("Repository name has not provided ")
			}

			repositoryName := args[0]

			repository, err := createRepository(repositoryName)
			if err != nil {
				return err
			}
			log.Infof("Repository %s/%s has been created successfully.", repository.Namespace, repository.Name)
			return err

		},
	}
	cmd.Flags().StringVar(&opt.Provider, "provider", opt.Provider, "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&opt.Bucket, "bucket", opt.Bucket, "Name of the cloud bucket")
	cmd.Flags().StringVar(&opt.Endpoint, "endpoint", opt.Endpoint, "Endpoint for s3/s3 compatible backend")
	cmd.Flags().StringVar(&opt.URL, "rest-server-url", opt.URL, "URL for rest backend")
	cmd.Flags().IntVar(&opt.MaxConnections, "max-connections", opt.MaxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")
	cmd.Flags().StringVar(&opt.Secret, "secret", opt.Secret, "Name of the Storage Secret")
	cmd.Flags().StringVar(&opt.Container, "container", opt.Container, "name of the cloud container")
	cmd.Flags().StringVar(&opt.Prefix, "prefix", opt.Prefix, "Prefix denotes the directory inside the backend")

	return cmd
}

func createRepository(name string) (repository *v1alpha1.Repository, err error) {
	repository = repositoryObj(name)
	repository, _, err = util.CreateOrPatchRepository(stashClient.StashV1alpha1(),
		metav1.ObjectMeta{
			Name:      repository.Name,
			Namespace: repository.Namespace,
		},
		func(obj *v1alpha1.Repository) *v1alpha1.Repository {
			obj.TypeMeta = repository.TypeMeta
			obj.Spec = repository.Spec
			return obj
		},
	)
	return repository, err
}

func repositoryObj(name string) *v1alpha1.Repository {

	backend := v1.Backend{}
	switch opt.Provider {
	case restic.ProviderGCS:
		backend = v1.Backend{
			StorageSecretName: opt.Secret,
			GCS: &v1.GCSSpec{
				Bucket:         opt.Bucket,
				Prefix:         opt.Prefix,
				MaxConnections: opt.MaxConnections,
			},
		}
	case restic.ProviderAzure:
		backend = v1.Backend{
			StorageSecretName: opt.Secret,
			Azure: &v1.AzureSpec{
				Container:      opt.Container,
				Prefix:         opt.Prefix,
				MaxConnections: opt.MaxConnections,
			},
		}
	case restic.ProviderS3:
		backend = v1.Backend{
			StorageSecretName: opt.Secret,
			S3: &v1.S3Spec{
				Bucket:   opt.Bucket,
				Prefix:   opt.Prefix,
				Endpoint: opt.Endpoint,
			},
		}
	case restic.ProviderB2:
		backend = v1.Backend{
			StorageSecretName: opt.Secret,
			B2: &v1.B2Spec{
				Bucket:         opt.Bucket,
				Prefix:         opt.Prefix,
				MaxConnections: opt.MaxConnections,
			},
		}
	case restic.ProviderSwift:
		backend = v1.Backend{
			StorageSecretName: opt.Secret,
			Swift: &v1.SwiftSpec{
				Container: opt.Container,
				Prefix:    opt.Prefix,
			},
		}
	case restic.ProviderRest:
		backend = v1.Backend{
			StorageSecretName: opt.Secret,
			Rest: &v1.RestServerSpec{
				URL: opt.URL,
			},
		}

	}
	return &v1alpha1.Repository{
		TypeMeta: metav1.TypeMeta{
			Kind: v1alpha1.ResourceKindRepository,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.RepositorySpec{
			Backend: backend,
		},
	}
}
