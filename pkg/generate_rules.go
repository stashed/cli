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
	"fmt"
	"os/exec"
	"strings"
	"time"

	"stash.appscode.dev/apimachinery/pkg/restic"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

type snapshotStat struct {
	name       string
	id         string
	repository string
	hostname   string
	createdAt  time.Time
}

type restoreRule struct {
	TargetHosts []string `json:"targetHosts,omitempty"`
	Snapshots   []string `json:"snapshots,omitempty"`
}

func NewCmdGenRules() *cobra.Command {
	var (
		requestTimeout string
		timestamp      string
	)

	cmd := &cobra.Command{
		Use:               "rules",
		Short:             `Generate restore rules from nearest snapshots at a specific time.`,
		Long:              `Generate restore rules for a repository based on the closest snapshots to a specific time`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("repository name not found")
			}
			repositoryName := args[0]

			if timestamp == "" {
				return fmt.Errorf("timestamp not found")
			}
			targetTime, err := time.Parse(time.RFC3339, timestamp)
			if err != nil {
				return fmt.Errorf("failed to parse timestamp: %w", err)
			}

			arguments := []string{"get", "snapshots", "-n", namespace, "-l", fmt.Sprintf("repository=%s", repositoryName)}
			if requestTimeout != "" {
				arguments = append(arguments, "--request-timeout", requestTimeout)
			}

			output, err := exec.Command("kubectl", arguments...).CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to execute kubectl command: %w", err)
			}

			lines := strings.Split(string(output), "\n")
			var snapshots []snapshotStat
			var nearestSnapshot snapshotStat
			nearestTimeDiff := time.Duration(0)

			// Parse each line into a Snapshot struct and find the nearest snapshots
			for _, line := range lines[1:] { // Start from the second line to skip the header
				fields := strings.Fields(line)
				if len(fields) == 5 {
					name := fields[0]
					id := fields[1]
					repository := fields[2]
					hostname := fields[3]
					createdAtStr := fields[4]

					createdAt, err := time.Parse(time.RFC3339, createdAtStr)
					if err != nil {
						return fmt.Errorf("failed to parse creation timestamp: %w", err)
					}
					snapshot := snapshotStat{
						name:       name,
						id:         id,
						repository: repository,
						hostname:   hostname,
						createdAt:  createdAt,
					}
					snapshots = append(snapshots, snapshot)

					if hostname != restic.DefaultHost {
						continue
					}

					// Calculate the absolute time difference
					timeDiff := targetTime.Sub(createdAt)
					if timeDiff < 0 {
						timeDiff = -timeDiff // Make sure it's positive
					}

					if timeDiff < nearestTimeDiff || nearestSnapshot.name == "" {
						nearestTimeDiff = timeDiff
						nearestSnapshot = snapshot
					}
				}
			}

			if len(snapshots) == 0 {
				return fmt.Errorf("no snapshots found")
			}

			numberOfHosts := getNumberOfHosts(snapshots)
			if err := validateSnapshots(snapshots, numberOfHosts); err != nil {
				return err
			}
			targetSnapshots := getTargetGroupOfSnapshots(snapshots, nearestSnapshot, numberOfHosts)
			if len(targetSnapshots) == 0 {
				return fmt.Errorf("failed to find snapshots for hosts")
			}

			rules := getRules(targetSnapshots)
			yamlData, err := yaml.Marshal(rules)
			if err != nil {
				return err
			}
			klog.V(0).Infoln("Rules for RestoreSession:", "\n", string(yamlData))
			return nil
		},
	}

	cmd.Flags().StringVar(&requestTimeout, "request-timeout", requestTimeout, "Request timeout duration for the kubectl command")
	cmd.Flags().StringVar(&timestamp, "timestamp", timestamp, "Timestamp to find the closest snapshots")

	return cmd
}

func getNumberOfHosts(snapshots []snapshotStat) int {
	uniqueHosts := make(map[string]struct{})
	for _, snap := range snapshots {
		uniqueHosts[snap.hostname] = struct{}{}
	}
	return len(uniqueHosts)
}

func validateSnapshots(snapshots []snapshotStat, numberOfHosts int) error {
	if len(snapshots)%numberOfHosts != 0 {
		return fmt.Errorf("one or more snapshots missing for one or more hosts")
	}

	for i := 0; i < len(snapshots); i += numberOfHosts {
		uniqueHosts := make(map[string]struct{})
		for j := i; j < i+numberOfHosts; j++ {
			uniqueHosts[snapshots[j].hostname] = struct{}{}
		}
		if len(uniqueHosts) != numberOfHosts {
			return fmt.Errorf("one or more snapshots missing for one or more hosts")
		}
	}
	return nil
}

func getRules(snapshots []snapshotStat) []restoreRule {
	if len(snapshots) == 1 {
		return []restoreRule{
			{
				Snapshots: []string{snapshots[0].id},
			},
		}
	}

	var rules []restoreRule
	for _, snap := range snapshots {
		rule := restoreRule{
			TargetHosts: []string{snap.hostname},
			Snapshots:   []string{snap.id},
		}
		rules = append(rules, rule)
	}
	return rules
}

func getTargetGroupOfSnapshots(snapshots []snapshotStat, nearestSnapshot snapshotStat, numberOfHosts int) []snapshotStat {
	nearestSnapshotIndex := findNearestSnapshotIndex(snapshots, nearestSnapshot)

	groupStartIndex := (nearestSnapshotIndex / numberOfHosts) * numberOfHosts

	return getSnapshotsInRange(snapshots, groupStartIndex, numberOfHosts)
}

func findNearestSnapshotIndex(snapshots []snapshotStat, nearestSnapshot snapshotStat) int {
	for i, snap := range snapshots {
		if snap.id == nearestSnapshot.id {
			return i
		}
	}
	return -1
}

func getSnapshotsInRange(snapshots []snapshotStat, startIndex, count int) []snapshotStat {
	if startIndex < 0 || startIndex >= len(snapshots) || count <= 0 {
		return nil
	}

	endIndex := startIndex + count
	if endIndex > len(snapshots) {
		return nil
	}

	return snapshots[startIndex:endIndex]
}
