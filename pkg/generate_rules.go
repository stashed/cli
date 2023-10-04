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
		requestTimeout           string
		timestamp                string
		snapshotGroupingInterval string
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

			output, err := exec.Command(cmdKubectl, arguments...).CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to execute kubectl command: %w", err)
			}
			groups, err := groupSnapshotsByTime(output, snapshotGroupingInterval)
			if err != nil {
				return err
			}

			if err := validateGroups(groups); err != nil {
				return err
			}

			closestBefore, closestAfter := findClosestGroups(groups, targetTime)

			if closestBefore != nil {
				rules := getRules(closestBefore)
				yamlData, err := yaml.Marshal(rules)
				if err != nil {
					return err
				}

				out := fmt.Sprintf("Rules determined by the closest snapshots preceding the given timestamp:\n%s", string(yamlData))
				klog.Infoln(out)
			}

			if closestAfter != nil {
				rules := getRules(closestAfter)
				yamlData, err := yaml.Marshal(rules)
				if err != nil {
					return err
				}

				out := fmt.Sprintf("Rules determined by the closest snapshots following or matching the given timestamp:\n%s", string(yamlData))
				klog.Infoln(out)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&requestTimeout, "request-timeout", requestTimeout, "Request timeout duration for the kubectl command")
	cmd.Flags().StringVar(&timestamp, "timestamp", timestamp, "Timestamp to find the closest snapshots")
	cmd.Flags().StringVar(&snapshotGroupingInterval, "group-interval", "4m", "Snaspshot grouping interval")
	return cmd
}

func groupSnapshotsByTime(out []byte, interval string) ([][]snapshotStat, error) {
	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("no snapshot data found")
	}

	intervalDuration, err := time.ParseDuration(interval)
	if err != nil {
		return nil, err
	}

	var groups [][]snapshotStat
	var currentGroup []snapshotStat

	for _, line := range lines[1:] { // Start from the second line to skip the header
		if len(line) == 0 {
			continue
		}
		snapshot, err := parseSnapshot(line)
		if err != nil {
			return nil, err
		}

		if len(currentGroup) == 0 {
			currentGroup = []snapshotStat{snapshot}
			continue
		}

		// Check if the current group is empty or if the time difference exceeds the interval.
		if snapshot.createdAt.Sub(currentGroup[0].createdAt) > intervalDuration {
			groups = append(groups, currentGroup)
			currentGroup = []snapshotStat{snapshot}
		} else {
			currentGroup = append(currentGroup, snapshot)
		}
	}

	if len(currentGroup) != 0 {
		groups = append(groups, currentGroup)
	}

	return groups, nil
}

func validateGroups(groups [][]snapshotStat) error {
	for _, snapshots := range groups {
		hostNames := make(map[string]struct{})
		for _, snapshot := range snapshots {
			if _, exists := hostNames[snapshot.hostname]; exists {
				return fmt.Errorf("invalid snapshots group: two snapshots of the same host found in the same group")
			}
			hostNames[snapshot.hostname] = struct{}{}
		}
	}
	return nil
}

func parseSnapshot(line string) (snapshotStat, error) {
	fields := strings.Fields(line)
	if len(fields) != 5 {
		return snapshotStat{}, fmt.Errorf("invalid snapshot line: %s", line)
	}
	name, id, repository, hostname, createdAtStr := fields[0], fields[1], fields[2], fields[3], fields[4]
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return snapshotStat{}, fmt.Errorf("failed to parse creation timestamp: %w", err)
	}

	return snapshotStat{
		name:       name,
		id:         id,
		repository: repository,
		hostname:   hostname,
		createdAt:  createdAt,
	}, nil
}

func findClosestGroups(groups [][]snapshotStat, targetTime time.Time) (closestBefore []snapshotStat, closestAfter []snapshotStat) {
	var shortestTimeDiffBefore, shortestTimeDiffAfter time.Duration

	for _, group := range groups {
		if len(group) == 0 {
			continue
		}

		groupTime := group[0].createdAt
		// Calculate the absolute time difference
		timeDiff := targetTime.Sub(groupTime)
		if timeDiff < 0 {
			timeDiff = -timeDiff // Make sure it's positive
		}

		if groupTime.Before(targetTime) && (closestBefore == nil || timeDiff < shortestTimeDiffBefore) {
			shortestTimeDiffBefore = timeDiff
			closestBefore = group
		} else if (groupTime.After(targetTime) || groupTime.Equal(targetTime)) && (closestAfter == nil || timeDiff < shortestTimeDiffAfter) {
			shortestTimeDiffAfter = timeDiff
			closestAfter = group
		}
	}
	return
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
