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

	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1alpha1/util"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/util/templates"
	storage "kmodules.xyz/objectstore-api/api/v1"
)

var createRepositoryExample = templates.Examples(`
		# Create a new repository
		stash create repository --namespace=<namespace> <repository-name> [Flag]
        stash create repository gcs-repo --namespace=demo --secret=gcs-secret --bucket=appscode-qa --prefix=/source/data --provider=gcs`)

type repositoryOption struct {
	provider       string
	bucket         string
	endpoint       string
	maxConnections int64
	secret         string
	prefix         string
}

func NewCmdCreateRepository() *cobra.Command {
	repoOpt := repositoryOption{}
	cmd := &cobra.Command{
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
			klog.Infof("Repository %s/%s has been created successfully.", repository.Namespace, repository.Name)
			return err
		},
	}
	cmd.Flags().StringVar(&repoOpt.provider, "provider", repoOpt.provider, "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&repoOpt.bucket, "bucket", repoOpt.bucket, "Name of the cloud bucket/container")
	cmd.Flags().StringVar(&repoOpt.endpoint, "endpoint", repoOpt.endpoint, "Endpoint for s3/s3 compatible backend")
	cmd.Flags().Int64Var(&repoOpt.maxConnections, "max-connections", repoOpt.maxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")
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
	repository, _, err := util.CreateOrPatchRepository(
		context.TODO(),
		stashClient.StashV1alpha1(),
		meta, func(in *v1alpha1.Repository) *v1alpha1.Repository {
			in.Spec = repository.Spec
			return in
		},
		metav1.PatchOptions{},
	)
	return repository, err
}

func (opt repositoryOption) getBackendInfo() storage.Backend {
	var backend storage.Backend
	switch opt.provider {
	case storage.ProviderGCS:
		backend = storage.Backend{
			GCS: &storage.GCSSpec{
				Bucket:         opt.bucket,
				Prefix:         opt.prefix,
				MaxConnections: opt.maxConnections,
			},
		}
	case storage.ProviderAzure:
		backend = storage.Backend{
			Azure: &storage.AzureSpec{
				Container:      opt.bucket,
				Prefix:         opt.prefix,
				MaxConnections: opt.maxConnections,
			},
		}
	case storage.ProviderS3:
		backend = storage.Backend{
			S3: &storage.S3Spec{
				Bucket:   opt.bucket,
				Prefix:   opt.prefix,
				Endpoint: opt.endpoint,
			},
		}
	case storage.ProviderB2:
		backend = storage.Backend{
			B2: &storage.B2Spec{
				Bucket:         opt.bucket,
				Prefix:         opt.prefix,
				MaxConnections: opt.maxConnections,
			},
		}
	case storage.ProviderSwift:
		backend = storage.Backend{
			Swift: &storage.SwiftSpec{
				Container: opt.bucket,
				Prefix:    opt.prefix,
			},
		}
	case storage.ProviderRest:
		backend = storage.Backend{
			Rest: &storage.RestServerSpec{
				URL: opt.endpoint,
			},
		}
	}
	backend.StorageSecretName = opt.secret
	return backend
}
