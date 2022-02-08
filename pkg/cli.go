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

	"stash.appscode.dev/apimachinery/pkg/docker"
)

const (
	configDirName = "config"
	ResticEnvs    = "restic-envs"
)

type cliLocalDirectories struct {
	configDir   string // temp dir
	downloadDir string // user provided or, current working dir
}

// These variables will be set during build time
const (
	ScratchDir     = "/tmp/scratch"
	DestinationDir = "/tmp/destination"
)

var (
	ResticRegistry = "stashed"
	ResticImage    = "restic"
	ResticTag      = "latest"
)

var (
	backupConfig string
	backupBatch  string
)

var imgRestic docker.Docker

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
	return os.MkdirAll(localDirs.downloadDir, 0755)
}
