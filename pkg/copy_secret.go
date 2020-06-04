/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

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

	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
)

func NewCmdCopySecret() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "secret",
		Short:             `Copy Secret`,
		Long:              `Copy Secret from one namespace to another namespace`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("secret name is not provided")
			}

			secretName := args[0]

			// get source secret from current namespace
			// if found then copy the secret to the destination namespace
			return ensureSecret(secretName)
		},
	}

	return cmd
}

func ensureSecret(name string) error {
	// get source Secret
	secret, err := kubeClient.CoreV1().Secrets(srcNamespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	log.Infof("Copying Storage Secret %s to %s namespace", secret.Namespace, dstNamespace)
	// copy the Secret to the destination namespace
	meta := metav1.ObjectMeta{
		Name:        secret.Name,
		Namespace:   dstNamespace,
		Labels:      secret.Labels,
		Annotations: secret.Annotations,
	}
	_, err = createSecret(secret, meta)
	if err != nil {
		return err
	}

	log.Infof("Secret %s/%s has been copied to %s namespace successfully.", secret.Namespace, secret.Name, dstNamespace)
	return err
}

func createSecret(secret *core.Secret, meta metav1.ObjectMeta) (*core.Secret, error) {
	secret, _, err := core_util.CreateOrPatchSecret(context.TODO(), kubeClient, meta, func(in *core.Secret) *core.Secret {
		in.Data = secret.Data
		return in
	}, metav1.PatchOptions{})
	return secret, err
}
