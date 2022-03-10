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
	"os"

	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/docker"

	vs_cs "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

// These variables will be set during build time
const (
	ScratchDir     = "/tmp/scratch"
	DestinationDir = "/tmp/destination"
	configDirName  = "config"

	ResticEnvs     = "restic-envs"
	ResticRegistry = "stashed"
	ResticImage    = "restic"
	ResticTag      = "latest"
)

type cliLocalDirectories struct {
	configDir   string // temp dir
	downloadDir string // user provided or, current working dir
}

var (
	dstNamespace string
	srcNamespace string
	namespace    string

	kubeClient  *kubernetes.Clientset
	stashClient *cs.Clientset
	aggrClient  *clientset.Clientset
	vsClient    *vs_cs.Clientset
	imgRestic   docker.Docker

	backupConfig   string
	backupBatch    string
	restoreSession string
	restoreBatch   string
)

func init() {
	imgRestic.Registry = ResticRegistry
	imgRestic.Image = ResticImage
	imgRestic.Tag = ResticTag
}

func (localDirs *cliLocalDirectories) prepareDownloadDir() (err error) {
	// if destination flag is not specified, restore in current directory
	if localDirs.downloadDir == "" {
		if localDirs.downloadDir, err = os.Getwd(); err != nil {
			return err
		}
	}
	return os.MkdirAll(localDirs.downloadDir, 0o755)
}
