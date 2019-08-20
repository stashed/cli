package pkg

import (
	"io/ioutil"
	core "k8s.io/api/core/v1"
	"os"
	"path/filepath"
	"stash.appscode.dev/cli/pkg/docker"
	docker_image "stash.appscode.dev/stash/pkg/docker"
	"stash.appscode.dev/stash/pkg/restic"
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

//write repository secret credential in a sub-dir inside tempDir
func (localDirs *cliLocalDirectories) dumpSecret(temDir string, secret *core.Secret) error {
	localDirs.secretDir = filepath.Join(temDir, secretDirName)
	if err := os.MkdirAll(localDirs.secretDir, 0755); err != nil {
		return err
	}

	for key, val := range secret.Data {
		if err := ioutil.WriteFile(filepath.Join(localDirs.secretDir,key), []byte(val), 0755); err != nil {
			return err
		}
	}

	return nil
}

func (localDirs *cliLocalDirectories) prepareConfDir(temDir string, dataString string) error {
	localDirs.configDir = filepath.Join(temDir, configDirName)
	if err := os.MkdirAll(localDirs.configDir, 0755); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(localDirs.configDir, "env"), []byte(dataString), 0755); err != nil {
		return err
	}

	return nil
}
