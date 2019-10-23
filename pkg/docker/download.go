package docker

import (
	"path/filepath"

	"stash.appscode.dev/stash/pkg/restic"

	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
)

// RemoveIt!
// Deprecated
func NewDownloadCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "download-snapshots",
		Short:             `Download snapshots`,
		Long:              `Download contents of snapshots from Repository`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			setupOpt, err := ReadSetupOptionFromFile(filepath.Join(ConfigDir, SetupOptionsFile))
			if err != nil {
				return err
			}
			restoreOpt, err := ReadRestoreOptionFromFile(filepath.Join(ConfigDir, RestoreOptionsFile))
			if err != nil {
				return err
			}
			resticWrapper, err := restic.NewResticWrapper(*setupOpt)
			if err != nil {
				return err
			}
			// run restore
			if _, err = resticWrapper.RunRestore(*restoreOpt); err != nil {
				return err
			}
			log.Infof("Restore completed")
			return nil
		},
	}
	return cmd
}
