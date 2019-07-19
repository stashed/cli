package docker

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"stash.appscode.dev/stash/pkg/restic"
)

const (
	ScratchDir         = "/tmp/scratch"
	SecretDir          = "/tmp/secret"
	ConfigDir          = "/tmp/config"
	DestinationDir     = "/tmp/destination"
	SetupOptionsFile   = "setup.json"
	RestoreOptionsFile = "restore.json"
)

func WriteSetupOptionToFile(options *restic.SetupOptions, fileName string) error {
	jsonOutput, err := json.MarshalIndent(options, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(fileName, jsonOutput, 0755); err != nil {
		return err
	}
	return nil
}

func ReadSetupOptionFromFile(filename string) (*restic.SetupOptions, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	options := &restic.SetupOptions{}
	err = json.Unmarshal(data, options)
	if err != nil {
		return nil, err
	}

	return options, nil
}

func WriteRestoreOptionToFile(options *restic.RestoreOptions, fileName string) error {
	jsonOutput, err := json.MarshalIndent(options, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(fileName, jsonOutput, 0755); err != nil {
		return err
	}
	return nil
}

func ReadRestoreOptionFromFile(filename string) (*restic.RestoreOptions, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	options := &restic.RestoreOptions{}
	err = json.Unmarshal(data, options)
	if err != nil {
		return nil, err
	}

	return options, nil
}
