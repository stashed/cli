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

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func NewCmdCopyRepository() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "repository",
		Short:             `Copy Repository and Secret`,
		Long:              `Copy Repository and Secret from one namespace to another namespace`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("Repository name is not provided")
			}

			repositoryName := args[0]
			// get source Repository from current namespace
			// if found then copy the Repository to the destination namespace
			return ensureRepository(repositoryName)
		},
	}

	return cmd
}

func ensureRepository(name string) error {
	// get source Repository
	repository, err := stashClient.StashV1alpha1().Repositories(srcNamespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	klog.Infof("Repository %s/%s uses Storage Secret %s/%s.", repository.Namespace, repository.Name, repository.Namespace, repository.Spec.Backend.StorageSecretName)
	// ensure source Repository Secret
	err = ensureSecret(repository.Spec.Backend.StorageSecretName)
	if err != nil {
		return err
	}
	// copy the Repository to the destination namespace
	meta := metav1.ObjectMeta{
		Name:        repository.Name,
		Namespace:   dstNamespace,
		Labels:      repository.Labels,
		Annotations: repository.Annotations,
	}
	_, err = createRepository(repository, meta)
	if err != nil {
		return err
	}
	klog.Infof("Repository %s/%s has been copied to %s namespace successfully.", repository.Namespace, repository.Name, dstNamespace)
	return err
}
