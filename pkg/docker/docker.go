package docker

import (
	"encoding/json"
	"io/ioutil"

	"stash.appscode.dev/stash/pkg/restic"
)

const (
	ScratchDir         = "/tmp/scratch"
	ConfigDir          = "/tmp/config"
	SetupOptionsFile   = "setup.json"
	RestoreOptionsFile = "restore.json"
)

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
