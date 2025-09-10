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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerr "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/util/templates"
	cu "kmodules.xyz/client-go/client"
	v1 "kmodules.xyz/objectstore-api/api/v1"
	kubestashapi "kubestash.dev/apimachinery/apis/storage/v1alpha1"
	"kubestash.dev/apimachinery/pkg/blob"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	ScriptPermissions     = 0o755
	ScratchDirPermissions = 0o755

	// TableMinWidth Output formatting
	TableMinWidth = 0
	TableTabWidth = 0
	TablePadding  = 2
	TablePadChar  = ' '

	// OutputTimeFormat Time formats
	OutputTimeFormat = "2006-01-02 15:04:05"
	LogTimeFormat    = time.RFC3339
)

type purgeOptions struct {
	// Kubernetes clients
	kubeClient *kubernetes.Clientset
	klient     client.Client
	config     *rest.Config

	// Configuration
	configFile    string // Path to backend config file
	backendConfig *v1.Backend

	// Command options
	olderThan string
	dryRun    bool

	// Runtime objects
	storage *blob.Blob
}

type repositoryInfo struct {
	Path         string
	LastModified time.Time
	Size         int64
}

type PurgeStats struct {
	TotalFound   int
	TotalDeleted int
	TotalFailed  int
	TotalSkipped int
	StartTime    time.Time
	EndTime      time.Time
	Errors       []error
}

func NewCmdPurgeRepos(clientGetter genericclioptions.RESTClientGetter) *cobra.Command {
	opt := purgeOptions{}
	cmd := &cobra.Command{
		Use:               "purge-repos",
		Short:             `Purge old repositories from backend storage`,
		DisableAutoGenTag: true,
		Example:           purgeExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opt.configFile == "" {
				return fmt.Errorf("--config-file flag is required. Provide a YAML/JSON file describing the backend")
			}
			if opt.olderThan == "" {
				return fmt.Errorf("--older-than flag is required. Example: 1y, 1y6mo, 1y6mo30d")
			}

			var err error
			opt.config, err = clientGetter.ToRESTConfig()
			if err != nil {
				return errors.Wrap(err, "failed to read kubeconfig")
			}
			namespace, _, err = clientGetter.ToRawKubeConfigLoader().Namespace()
			if err != nil {
				return fmt.Errorf("failed to get namespace: %w", err)
			}

			opt.klient, err = newUncachedClient()
			if err != nil {
				return err
			}

			opt.kubeClient, err = kubernetes.NewForConfig(opt.config)
			if err != nil {
				return err
			}

			return opt.purgeRepositories()
		},
	}

	cmd.Flags().StringVar(&opt.configFile, "backend-config", "", "Path to backend config YAML/JSON file (required)")
	cmd.Flags().StringVar(&opt.olderThan, "older-than", "", "Purge repositories older than this duration (e.g., 1y, 6mo, 30d, 24h)")
	cmd.Flags().BoolVar(&opt.dryRun, "dry-run", false, "List repositories that would be deleted without actually deleting them")

	return cmd
}

func (opt *purgeOptions) purgeRepositories() error {
	var err error
	if opt.backendConfig, err = opt.validateAndLoadConfig(); err != nil {
		return err
	}
	if opt.storage, err = opt.getBlobStorageFromConfig(); err != nil {
		return err
	}
	cutoffTime, err := opt.parseDurationWithValidation()
	if err != nil {
		return err
	}

	opt.logOperationDetails(cutoffTime)
	if err := opt.setupScratchDirectory(); err != nil {
		return err
	}
	defer opt.cleanupScratchDirectory()

	secret, err := opt.getStorageSecret()
	if err != nil {
		return err
	}
	setupOpt, err := opt.setupResticWrapper(secret)
	if err != nil {
		return err
	}
	return opt.executePurgeWorkflow(setupOpt, cutoffTime)
}

func (opt *purgeOptions) validateAndLoadConfig() (*v1.Backend, error) {
	if opt.configFile == "" {
		return nil, fmt.Errorf("--config-file flag is required. Provide a YAML/JSON file describing the backend")
	}
	if opt.olderThan == "" {
		return nil, fmt.Errorf("--older-than flag is required. Example: 1y, 1y6mo, 1y6mo30d")
	}
	cfg, err := loadBackendConfig(opt.configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load backend config: %v", err)
	}
	return cfg, nil
}

func (opt *purgeOptions) parseDurationWithValidation() (time.Time, error) {
	cutoffTime, err := opt.parseDuration()
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid duration format: %s", opt.olderThan)
	}
	return cutoffTime, nil
}

func (opt *purgeOptions) logOperationDetails(cutoffTime time.Time) {
	klog.Infof("Starting repository purge operation")
	klog.Infof("Configuration file: %s", opt.configFile)
	klog.Infof("duration filter: %s (cutoff: %s)", opt.olderThan, cutoffTime.Format(LogTimeFormat))
	klog.Infof("Dry run mode: %t\n", opt.dryRun)
}

func (opt *purgeOptions) getStorageSecret() (*core.Secret, error) {
	if opt.backendConfig.StorageSecretName == "" {
		return nil, fmt.Errorf("storageSecretName is required in backend configuration")
	}

	secret, err := opt.kubeClient.CoreV1().Secrets(namespace).Get(
		context.TODO(),
		opt.backendConfig.StorageSecretName,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %v", namespace, opt.backendConfig.StorageSecretName, err)
	}

	return secret, nil
}

func (opt *purgeOptions) setupScratchDirectory() error {
	if err := os.MkdirAll(ScratchDir, ScratchDirPermissions); err != nil {
		return fmt.Errorf("failed to create scratch directory: %v", err)
	}
	return nil
}

func (opt *purgeOptions) cleanupScratchDirectory() {
	if err := os.RemoveAll(ScratchDir); err != nil {
		klog.Warningf("Failed to cleanup scratch directory: %v", err)
	}
}

func (opt *purgeOptions) setupResticWrapper(secret *core.Secret) (restic.SetupOptions, error) {
	extraOpt := util.ExtraOptions{
		StorageSecret: secret,
		ScratchDir:    ScratchDir,
	}
	tempRepo := &v1alpha1.Repository{
		Spec: v1alpha1.RepositorySpec{
			Backend: *opt.backendConfig,
		},
	}
	setupOpt, err := util.SetupOptionsForRepository(*tempRepo, extraOpt)
	if err != nil {
		return setupOpt, fmt.Errorf("failed to setup restic wrapper: %v", err)
	}
	return setupOpt, nil
}

func (opt *purgeOptions) executePurgeWorkflow(setupOpt restic.SetupOptions, cutoffTime time.Time) error {
	repoList, err := opt.findRepositoriesToPurge(setupOpt, cutoffTime)
	if err != nil {
		return err
	}

	if len(repoList) == 0 {
		opt.displayNoRepositoriesMessage()
		return nil
	}

	// Get repository base URL for display purposes
	repoBase, err := opt.getResticRepoFromEnv(setupOpt)
	if err != nil {
		return fmt.Errorf("failed to get restic repository base: %w", err)
	}

	opt.displayRepositoriesTable(repoList, repoBase)
	if opt.dryRun {
		displayDryRunMessage(len(repoList))
		return nil
	}

	if !opt.confirmDeletion(len(repoList)) {
		fmt.Println("Operation cancelled.")
		return nil
	}
	return opt.deleteRepositories(setupOpt, repoList)
}

func (opt *purgeOptions) findRepositoriesToPurge(setupOpt restic.SetupOptions, cutoffTime time.Time) ([]repositoryInfo, error) {
	var repos []repositoryInfo
	subDirs, err := opt.listSubdirectories("")
	if err != nil {
		return nil, fmt.Errorf("cannot list sub-dirs: %w", err)
	}

	repoBase, err := opt.getResticRepoFromEnv(setupOpt)
	if err != nil {
		return nil, err
	}

	script := opt.generateRepoListScript(repoBase, subDirs)
	out, err := runResticScriptViaDocker(script)
	if err != nil {
		return nil, fmt.Errorf("Error running repo check script: %v\nOutput:\n%s", err, out)
	}

	err = extractRepoListFromOutput(out, subDirs, cutoffTime, &repos)
	if err != nil {
		return nil, err
	}
	return repos, nil
}

func (opt *purgeOptions) getResticRepoFromEnv(setupOpt restic.SetupOptions) (string, error) {
	localDirs := &cliLocalDirectories{
		configDir: filepath.Join(ScratchDir, configDirName),
	}
	rw, err := restic.NewResticWrapper(setupOpt)
	if err != nil {
		return "", fmt.Errorf("failed to create restic wrapper: %v", err)
	}
	if err := rw.DumpEnv(localDirs.configDir, ResticEnvs); err != nil {
		return "", fmt.Errorf("failed to dump env: %v", err)
	}
	envData, err := os.ReadFile(filepath.Join(ScratchDir, configDirName, ResticEnvs))
	if err != nil {
		return "", fmt.Errorf("failed to read env file: %v", err)
	}

	var repoBase string
	for _, line := range strings.Split(string(envData), "\n") {
		if strings.HasPrefix(line, "RESTIC_REPOSITORY=") {
			repoBase = strings.TrimPrefix(line, "RESTIC_REPOSITORY=")
			repoBase = strings.Trim(repoBase, `"`)
			break
		}
	}
	if repoBase == "" {
		return "", fmt.Errorf("RESTIC_REPOSITORY not found in env file")
	}
	return repoBase, nil
}

func (opt *purgeOptions) generateRepoListScript(repoBase string, subDirs []string) string {
	var lines []string
	lines = append(lines, "#!/bin/sh")
	for _, dir := range subDirs {
		repoURL := strings.TrimRight(repoBase+"/"+dir, "/")
		lines = append(lines, fmt.Sprintf(
			`RESTIC_REPOSITORY="%s" restic snapshots --no-cache --latest 1 --json || echo "Failed to access repository %s"`, repoURL, dir))
	}
	return strings.Join(lines, "\n")
}

func (opt *purgeOptions) listSubdirectories(path string) ([]string, error) {
	entries, err := opt.storage.ListDirN(context.Background(), path)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))

	for _, raw := range entries {
		out = append(out, strings.TrimSuffix(string(raw), "/"))
	}
	return out, nil
}

func (opt *purgeOptions) deleteRepositories(setupOpt restic.SetupOptions, repos []repositoryInfo) error {
	stats := &PurgeStats{
		TotalFound: len(repos),
		StartTime:  time.Now(),
	}
	defer func() {
		stats.EndTime = time.Now()
		opt.displayPurgeStats(stats)
	}()

	repoBase, err := opt.getResticRepoFromEnv(setupOpt)
	if err != nil {
		return fmt.Errorf("failed to get restic repo from env: %w", err)
	}

	// Execute restic purge operations
	fmt.Println("Starting repository deletion process...")
	script := opt.generateRepoPurgeScript(repoBase, repos)
	out, err := runResticScriptViaDocker(script)
	if err != nil {
		return fmt.Errorf("failed to execute restic purge script: %w\nOutput:\n%s", err, out)
	}

	fmt.Printf("\n===== Snapshot Deletion Summary =====\n%s\n", out)

	// Clean up storage metadata
	fmt.Println("Cleaning up storage metadata...")
	prefix, err := opt.backendConfig.Prefix()
	if err != nil {
		return fmt.Errorf("failed to get prefix from backend config: %w", err)
	}

	for _, repo := range repos {
		repoURL := strings.TrimRight(repoBase+"/"+repo.Path, "/")
		if err := opt.deleteRepositoryMetadata(repo, prefix); err != nil {
			fmt.Printf("❌ %s: metadata not deleted\n", repoURL)
			stats.TotalFailed++
			stats.Errors = append(stats.Errors, err)
		} else {
			fmt.Printf("✅ %s: metadata deleted\n", repoURL)
			stats.TotalDeleted++
		}
	}

	if stats.TotalFailed > 0 {
		return fmt.Errorf("failed to delete %d out of %d repositories", stats.TotalFailed, stats.TotalFound)
	}

	return nil
}

func (opt *purgeOptions) deleteRepositoryMetadata(repo repositoryInfo, prefix string) error {
	repoPath := strings.Trim(repo.Path, "/")
	suffix := strings.TrimPrefix(repoPath, strings.Trim(prefix, "/")+"/")

	// Special case: if repoPath == prefix only
	if suffix == prefix {
		suffix = ""
	}

	if err := opt.storage.Delete(context.Background(), suffix, true); err != nil {
		return fmt.Errorf("failed to delete storage metadata for %s: %w", repo.Path, err)
	}

	klog.V(2).Infof("Successfully deleted metadata for repository: %s", repo.Path)
	return nil
}

func (opt *purgeOptions) displayPurgeStats(stats *PurgeStats) {
	fmt.Printf("\n===== Final Summary =====\n")
	fmt.Printf("Operation completed in %v\n", stats.duration())
	fmt.Printf("Successfully deleted: %d repositories\n", stats.TotalDeleted)

	if stats.TotalFailed > 0 {
		fmt.Printf("Failed to delete: %d repositories\n", stats.TotalFailed)
	}

	if stats.TotalSkipped > 0 {
		fmt.Printf("Skipped: %d repositories\n", stats.TotalSkipped)
	}

	successRate := float64(stats.TotalDeleted) / float64(stats.TotalFound) * 100
	fmt.Printf("Success rate: %.1f%%\n", successRate)
}

func (opt *purgeOptions) getBlobStorageFromConfig() (*blob.Blob, error) {
	bs := &kubestashapi.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: kubestashapi.BackupStorageSpec{},
	}

	// Configure storage based on backend type
	switch {
	case opt.backendConfig.S3 != nil:
		bs.Spec.Storage.Provider = kubestashapi.ProviderS3
		bs.Spec.Storage.S3 = &kubestashapi.S3Spec{
			Endpoint:    opt.backendConfig.S3.Endpoint,
			Bucket:      opt.backendConfig.S3.Bucket,
			Prefix:      opt.backendConfig.S3.Prefix,
			Region:      opt.backendConfig.S3.Region,
			SecretName:  opt.backendConfig.StorageSecretName,
			InsecureTLS: opt.backendConfig.S3.InsecureTLS,
		}

	case opt.backendConfig.Azure != nil:
		bs.Spec.Storage.Provider = kubestashapi.ProviderAzure
		bs.Spec.Storage.Azure = &kubestashapi.AzureSpec{
			Container:      opt.backendConfig.Azure.Container,
			Prefix:         opt.backendConfig.Azure.Prefix,
			SecretName:     opt.backendConfig.StorageSecretName,
			MaxConnections: opt.backendConfig.Azure.MaxConnections,
		}

	case opt.backendConfig.GCS != nil:
		bs.Spec.Storage.Provider = kubestashapi.ProviderGCS
		bs.Spec.Storage.GCS = &kubestashapi.GCSSpec{
			Bucket:         opt.backendConfig.GCS.Bucket,
			Prefix:         opt.backendConfig.GCS.Prefix,
			SecretName:     opt.backendConfig.StorageSecretName,
			MaxConnections: opt.backendConfig.GCS.MaxConnections,
		}
	default:
		return nil, fmt.Errorf("no storage backend configured. Reason: Unknown backend type")
	}

	storage, err := blob.NewBlob(context.Background(), opt.klient, bs)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob storage client: %w", err)
	}

	return storage, nil
}

func (opt *purgeOptions) parseDuration() (time.Time, error) {
	// Parse duration string like "1y", "6mo", "30d", "24h"
	durationRegex := regexp.MustCompile(`(\d+)([ydhms]|mo)`)
	matches := durationRegex.FindAllStringSubmatch(opt.olderThan, -1)

	if len(matches) == 0 {
		return time.Time{}, fmt.Errorf("invalid duration format: %s", opt.olderThan)
	}

	now := time.Now()
	cutoff := now

	for _, match := range matches {
		if len(match) != 3 {
			continue
		}

		value, err := strconv.Atoi(match[1])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid duration value: %s", match[1])
		}

		unit := match[2]
		switch unit {
		case "y":
			cutoff = cutoff.AddDate(-value, 0, 0)
		case "mo":
			cutoff = cutoff.AddDate(0, -value, 0)
		case "d":
			cutoff = cutoff.AddDate(0, 0, -value)
		case "h":
			cutoff = cutoff.Add(-time.Duration(value) * time.Hour)
		case "m":
			cutoff = cutoff.Add(-time.Duration(value) * time.Minute)
		case "s":
			cutoff = cutoff.Add(-time.Duration(value) * time.Second)
		default:
			return time.Time{}, fmt.Errorf("unsupported duration unit: %s", unit)
		}
	}

	return cutoff, nil
}

func (opt *purgeOptions) displayNoRepositoriesMessage() {
	fmt.Println("✅ No repositories found matching the criteria.")
	fmt.Printf("   - Age filter: older than %s\n", opt.olderThan)
}

func (opt *purgeOptions) displayRepositoriesTable(repos []repositoryInfo, repoBase string) {
	fmt.Printf("\nFound %d repositories to purge:\n", len(repos))

	// Create a tabwriter for formatted output
	w := tabwriter.NewWriter(os.Stdout, TableMinWidth, TableTabWidth, TablePadding, TablePadChar, 0)
	defer func() {
		_ = w.Flush() // Handle error silently for display purposes
	}()

	// Header - Updated to show "REPOSITORY" to match your desired output
	_, _ = fmt.Fprintf(w, "REPOSITORY\tLAST MODIFIED\tAGE\n")
	_, _ = fmt.Fprintf(w, "----------\t-------------\t---\n")

	// Data rows
	now := time.Now()
	for _, repo := range repos {
		age := now.Sub(repo.LastModified)
		ageStr := formatDuration(age)
		repoURL := strings.TrimRight(repoBase+"/"+repo.Path, "/")
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
			repoURL,
			repo.LastModified.Format(OutputTimeFormat),
			ageStr)
	}
	fmt.Println()
}

func (opt *purgeOptions) confirmDeletion(count int) bool {
	fmt.Printf("\nThis will permanently delete %d repositories. Are you sure? (y/N): ", count)
	var confirmation string
	_, _ = fmt.Scanln(&confirmation)
	confirmation = strings.ToLower(strings.TrimSpace(confirmation))
	return confirmation == "y" || confirmation == "yes"
}

func (opt *purgeOptions) generateRepoPurgeScript(repoBase string, repos []repositoryInfo) string {
	var lines []string
	lines = append(lines, "#!/bin/sh", "set -euo pipefail", "")
	lines = append(lines, "results=\"\"", "")
	lines = append(lines, `purge_repo() {
	  repo=$1
	  export RESTIC_REPOSITORY="$repo"

	  if ! restic forget --keep-last 1 --group-by '' --prune --no-cache --json >/dev/null 2>&1; then
		echo "Failed forget (keep-last) for $repo"
		results="$results\n❌ $repo: failed at keep-last"
		return 1
	  fi

	  ID=$(restic snapshots --latest 1 --no-cache --json | jq -r '.[0].id // empty')
	  if [ -z "$ID" ]; then
		echo "Repo $repo is already empty"
		results="$results\n⚠️  $repo: already empty"
		return 0
	  fi

	  if ! restic forget "$ID" --prune --no-cache >/dev/null 2>&1; then
		echo "Failed final forget for $repo"
		results="$results\n❌ $repo: failed at final forget"
		return 1
	  fi

	  if restic snapshots --json --no-cache | jq -e 'length==0' >/dev/null; then
		results="$results\n✅ $repo: all snapshots purged"
	  else
		echo "Repo $repo: some snapshots remain"
		results="$results\n⚠️  $repo: not fully purged"
	  fi
	}`)

	for _, repo := range repos {
		repoURL := strings.TrimRight(repoBase+"/"+repo.Path, "/")
		lines = append(lines, fmt.Sprintf(`purge_repo "%s"`, repoURL))
	}

	lines = append(lines, `echo -e "$results"`)
	return strings.Join(lines, "\n")
}

func (s *PurgeStats) duration() time.Duration {
	return s.EndTime.Sub(s.StartTime)
}

func extractRepoListFromOutput(out string, subDirs []string, cutoffTime time.Time, repos *[]repositoryInfo) error {
	type snapshot struct {
		Time string `json:"time"`
	}
	dirIndex := 0
	var errs []error
	var snapshots []snapshot
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip error messages and separators
		if strings.HasPrefix(line, "Failed to access repository") ||
			strings.Contains(line, "Fatal: repository does not exist") {
			// If we hit an error, we should still increment dirIndex to stay in sync
			if dirIndex < len(subDirs) {
				dirIndex++
			}
			continue
		}

		// Parse JSON array
		if strings.HasPrefix(line, "[") {
			if err := json.Unmarshal([]byte(line), &snapshots); err != nil {
				errs = append(errs, fmt.Errorf("failed to parse JSON for %s: %v", line, err))
				if dirIndex < len(subDirs) {
					dirIndex++
				}
				continue
			}

			if len(snapshots) > 0 {
				snapshotTime, err := time.Parse(time.RFC3339Nano, snapshots[0].Time)
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to parse time for %s: %v", line, err))
					if dirIndex < len(subDirs) {
						dirIndex++
					}
					continue
				}
				if dirIndex < len(subDirs) && snapshotTime.Before(cutoffTime) {
					*repos = append(*repos, repositoryInfo{
						Path:         subDirs[dirIndex],
						LastModified: snapshotTime,
					})
				}
			}
			dirIndex++
		}
	}

	return kerr.NewAggregate(errs)
}

func loadBackendConfig(path string) (*v1.Backend, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg v1.Backend
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

func newUncachedClient() (client.Client, error) {
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubernetes config: %w", err)
	}

	return cu.NewUncachedClient(
		cfg,
		clientsetscheme.AddToScheme,
	)
}

func runResticScriptViaDocker(script string) (string, error) {
	localDirs := &cliLocalDirectories{
		configDir: filepath.Join(ScratchDir, configDirName),
	}

	uid, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	scriptFile := filepath.Join(localDirs.configDir, "check.sh")
	if err := os.WriteFile(scriptFile, []byte(script), ScriptPermissions); err != nil {
		return "", fmt.Errorf("failed to write script file: %w", err)
	}

	args := []string{
		"run", "--rm",
		"-u", uid.Uid,
		"-v", ScratchDir + ":" + ScratchDir,
		"--env-file", filepath.Join(localDirs.configDir, ResticEnvs),
		"--entrypoint", "sh",
		imgRestic.ToContainerImage(),
		"-c", "/tmp/scratch/config/check.sh",
	}

	out, err := exec.Command("docker", args...).CombinedOutput()
	return string(out), err
}

func displayDryRunMessage(count int) {
	fmt.Printf("\nDry run completed. %d repositories would be deleted.\n", count)
	fmt.Println("To actually delete these repositories, run the command without --dry-run")
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 365 {
		years := days / 365
		remainingDays := days % 365
		if remainingDays > 0 {
			return fmt.Sprintf("%dy %dd", years, remainingDays)
		}
		return fmt.Sprintf("%dy", years)
	} else if days > 30 {
		months := days / 30
		remainingDays := days % 30
		if remainingDays > 0 {
			return fmt.Sprintf("%dmo %dd", months, remainingDays)
		}
		return fmt.Sprintf("%dmo", months)
	} else if days > 0 {
		if hours > 0 {
			return fmt.Sprintf("%dd %dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	} else if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	} else {
		return fmt.Sprintf("%dm", minutes)
	}
}

var purgeExample = templates.Examples(`
		# Basic usage: Purge repositories older than 1 year
		kubectl stash purge-repos --backend-config=backend.yaml --older-than=1y

		# Dry run to see what would be deleted without actually deleting
		kubectl stash purge-repos --backend-config=backend.yaml --older-than=6mo --dry-run

		# Purge with different time formats
		kubectl stash purge-repos --backend-config=backend.yaml --older-than=30d
		kubectl stash purge-repos --backend-config=backend.yaml --older-than=6mo
		kubectl stash purge-repos --backend-config=backend.yaml --older-than=1y6mo
		kubectl stash purge-repos --backend-config=backend.yaml --older-than=24h

`)
