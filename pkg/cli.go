package pkg

import (
	"fmt"
	"io/ioutil"
	core "k8s.io/api/core/v1"
	"os"
	"path/filepath"
	"stash.appscode.dev/cli/pkg/docker"
	docker_image "stash.appscode.dev/stash/pkg/docker"
	"stash.appscode.dev/stash/pkg/restic"
	"strings"
)

const (
	secretDirName = "secret"
	configDirName = "config"
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


func (localDirs *cliLocalDirectories) prepareDir(tempDir string, secret *core.Secret) error {
	var dataString string

	for key, value := range secret.Data {
		val := strings.ReplaceAll(string(value), "\n", "")

		if key == restic.GOOGLE_SERVICE_ACCOUNT_JSON_KEY {
			path := filepath.Join(localDirs.secretDir,restic.GOOGLE_APPLICATION_CREDENTIALS)
			if err := ioutil.WriteFile(path, []byte(val), 0755); err != nil {
				return err
			}
			val = path
			key = restic.GOOGLE_APPLICATION_CREDENTIALS
		}
		secretData := key + "=" + val
		dataString = dataString + fmt.Sprintln(secretData)
	}

	if err := ioutil.WriteFile(filepath.Join(localDirs.secretDir,"env"), []byte(dataString), 0755); err != nil {
		return err
	}

	return nil
}