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
	"io/ioutil"
	"os"
	"path/filepath"

	docker_image "stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/cli/pkg/docker"

	core "k8s.io/api/core/v1"
)

const (
	secretDirName = "secret"
	configDirName = "config"
	ResticEnvs    = "restic-envs"
)

type cliLocalDirectories struct {
	secretDir   string // temp dir
	configDir   string // temp dir
	downloadDir string // user provided or, current working dir
}

var (
	imgRestic = docker_image.Docker{
		Registry: "restic",
		Image:    "restic",
		Tag:      "0.9.5", // TODO: update default release tag
	}
)

func (localDirs *cliLocalDirectories) prepareSecretDir(tempDir string, secret *core.Secret) error {
	// write repository secrets in a sub-dir insider tempDir
	localDirs.secretDir = filepath.Join(tempDir, secretDirName)
	if err := os.MkdirAll(localDirs.secretDir, 0755); err != nil {
		return err
	}
	for key, value := range secret.Data {
		if err := ioutil.WriteFile(filepath.Join(localDirs.secretDir, key), value, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (localDirs *cliLocalDirectories) prepareConfigDir(tempDir string, setupOpt *restic.SetupOptions, restoreOpt *restic.RestoreOptions) error {
	// write restic options in a sub-dir insider tempDir
	localDirs.configDir = filepath.Join(tempDir, configDirName)
	if err := os.MkdirAll(localDirs.secretDir, 0755); err != nil {
		return err
	}
	if setupOpt != nil {
		err := docker.WriteSetupOptionToFile(setupOpt, filepath.Join(localDirs.configDir, docker.SetupOptionsFile))
		if err != nil {
			return err
		}
	}
	if restoreOpt != nil {
		err := docker.WriteRestoreOptionToFile(restoreOpt, filepath.Join(localDirs.configDir, docker.RestoreOptionsFile))
		if err != nil {
			return err
		}
	}
	return nil
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

// Write Storage Secret credentials in secret dir inside tempDir
func (localDirs *cliLocalDirectories) dumpSecret(temDir string, secret *core.Secret) error {
	localDirs.secretDir = filepath.Join(temDir, secretDirName)
	if err := os.MkdirAll(localDirs.secretDir, 0755); err != nil {
		return err
	}

	for key, val := range secret.Data {
		if err := ioutil.WriteFile(filepath.Join(localDirs.secretDir, key), []byte(val), 0755); err != nil {
			return err
		}
	}

	return nil
}
