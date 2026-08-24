// Package installskills installs the reviewed gplay agent-skill pack without
// executing npm, npx, or any code from the skills repository.
package installskills

import (
	"context"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- Git tree object IDs are SHA-1 by definition.
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

const (
	skillsSourceRepositoryURL = "https://github.com/tamtom/gplay-cli-skills.git"
	skillsSourceCommit        = "10301b24639e4f768d009b2edda9315cb2149712"
	installLockName           = ".gplay-install-skills.lock"
)

type skillPin struct {
	Name     string `json:"name"`
	TreeHash string `json:"treeHash"`
}

var skillPins = []skillPin{
	{Name: "gplay-cli-usage", TreeHash: "dff86d5b9b9da8a9608b74c10a9da87587efc75e"},
	{Name: "gplay-gradle-build", TreeHash: "107f0a3d3b52b3059f301400f050391e926d892e"},
	{Name: "gplay-iap-setup", TreeHash: "2ffca2c7bfe12ccf0b1e0844dff0059111c0987a"},
	{Name: "gplay-metadata-sync", TreeHash: "ffa500a6aecb27db8215224b256cab55014b8890"},
	{Name: "gplay-migrate-fastlane", TreeHash: "802f4bbe9752dbe533395db3bf7befea98e2ff48"},
	{Name: "gplay-ppp-pricing", TreeHash: "6a4f16ca19e688c290fc1d35bca9f184216f66ec"},
	{Name: "gplay-preflight", TreeHash: "555a8f64392a639826f0e122afd1c8c1e3da1fb5"},
	{Name: "gplay-purchase-verification", TreeHash: "7c457e13da0968df902f7da5e49f32ef7b3b78ed"},
	{Name: "gplay-release-flow", TreeHash: "ba74b56b7b2bbd83e2ec54446649ff8cebe9a554"},
	{Name: "gplay-reports-download", TreeHash: "f367979760c36032af48e0e4ef554b8bb0ed6e29"},
	{Name: "gplay-review-management", TreeHash: "70f21e28eb252a884a984014dbd12626f9e3b343"},
	{Name: "gplay-rollout-management", TreeHash: "990f6634ac2d32901f54a787415256d52b44d6e7"},
	{Name: "gplay-screenshot-automation", TreeHash: "3d62dc0f93fbbb4655c9dbc1dec87c755aafdc83"},
	{Name: "gplay-submission-checks", TreeHash: "462a876538a29760b01f862bd6cc57f57e65e7a0"},
	{Name: "gplay-testers-orchestration", TreeHash: "4dfba1c98e05f0087a5500195f007ea741ab085d"},
	{Name: "gplay-user-management", TreeHash: "09f69885ad18d409b4718272495f615dbff5e11a"},
	{Name: "gplay-vitals-monitoring", TreeHash: "11acaf969dcf1f4a9469868535600d30930e25ee"},
}

var (
	userHomeDirectory    = os.UserHomeDir
	checkoutPinnedSkills = defaultCheckoutPinnedSkills
	renameWithinRoot     = func(root *os.Root, oldName, newName string) error { return root.Rename(oldName, newName) }
)

type installOptions struct {
	Destination string
	Preview     bool
	Force       bool
}

type skillInstallItem struct {
	Name     string `json:"name"`
	TreeHash string `json:"treeHash"`
	Action   string `json:"action"`
}

type skillInstallResult struct {
	Version          int                `json:"version"`
	Mode             string             `json:"mode"`
	SourceRepository string             `json:"sourceRepository"`
	SourceCommit     string             `json:"sourceCommit"`
	Destination      string             `json:"destination"`
	Force            bool               `json:"force"`
	InstalledCount   int                `json:"installedCount"`
	Skills           []skillInstallItem `json:"skills"`
}

// InstallSkillsCommand returns the top-level install-skills command.
func InstallSkillsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("install-skills", flag.ExitOnError)
	destination := fs.String("dest", "", "Skill destination (default: ~/.agents/skills)")
	preview := fs.Bool("preview", false, "Show the verified plan without downloading or writing")
	force := fs.Bool("force", false, "Replace existing gplay skills with rollback protection")
	output := fs.String("output", "json", "Output format: json, table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	longHelp := "Install the pinned gplay agent-skill pack without executing package installers.\n\n" +
		"The immutable reviewed commit is " + skillsSourceCommit + ". Every Git tree hash is verified. " +
		"Existing skills are preserved unless --force is explicit, and forced replacements roll back as a unit.\n\n" +
		"--preview and global --dry-run perform no network request and no filesystem write."

	return &ffcli.Command{
		Name:       "install-skills",
		ShortUsage: "gplay install-skills [--preview] [--force] [--dest <path>]",
		ShortHelp:  "Install the pinned, verified gplay agent-skill pack.",
		LongHelp:   longHelp,
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				return shared.UsageErrorf("unexpected arguments: %s", strings.Join(args, " "))
			}
			if *force && *preview {
				return shared.UsageError("--force and --preview cannot be used together")
			}
			result, err := install(ctx, installOptions{
				Destination: *destination,
				Preview:     *preview || shared.IsDryRun(ctx),
				Force:       *force,
			})
			if err != nil {
				return fmt.Errorf("install skills: %w", err)
			}
			return shared.PrintOutputContext(ctx, result, *output, *pretty)
		},
	}
}

func install(ctx context.Context, opts installOptions) (skillInstallResult, error) {
	destination, err := resolveDestination(opts.Destination)
	if err != nil {
		return skillInstallResult{}, err
	}
	pins := append([]skillPin(nil), skillPins...)
	sort.Slice(pins, func(i, j int) bool { return pins[i].Name < pins[j].Name })
	conflicts, err := findConflicts(destination, pins)
	if err != nil {
		return skillInstallResult{}, err
	}
	result := installationResult(destination, pins, conflicts, opts)
	if opts.Preview {
		return result, nil
	}
	if len(conflicts) > 0 && !opts.Force {
		return skillInstallResult{}, fmt.Errorf("skills already exist: %s; rerun with --force to replace them", strings.Join(conflicts, ", "))
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return skillInstallResult{}, fmt.Errorf("create skill destination: %w", err)
	}
	destinationRoot, err := os.OpenRoot(destination)
	if err != nil {
		return skillInstallResult{}, fmt.Errorf("open skill destination: %w", err)
	}
	defer func() { _ = destinationRoot.Close() }()

	lock, err := destinationRoot.OpenFile(installLockName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return skillInstallResult{}, fmt.Errorf("acquire install lock %q: %w", filepath.Join(destination, installLockName), err)
	}
	_, _ = fmt.Fprintf(lock, "pid=%d\n", os.Getpid())
	_ = lock.Sync()
	defer func() {
		_ = lock.Close()
		_ = destinationRoot.Remove(installLockName)
	}()

	workspace, err := os.MkdirTemp("", "gplay-install-skills-")
	if err != nil {
		return skillInstallResult{}, fmt.Errorf("create private checkout workspace: %w", err)
	}
	defer func() { _ = os.RemoveAll(workspace) }()
	if err := os.Chmod(workspace, 0o700); err != nil {
		return skillInstallResult{}, fmt.Errorf("protect checkout workspace: %w", err)
	}
	source, err := checkoutPinnedSkills(ctx, workspace)
	if err != nil {
		return skillInstallResult{}, err
	}
	if err := validatePinnedPack(source, pins); err != nil {
		return skillInstallResult{}, err
	}
	if err := installPack(source, destinationRoot, pins, opts.Force); err != nil {
		return skillInstallResult{}, err
	}
	result.Mode = "installed"
	result.InstalledCount = len(pins)
	for i := range result.Skills {
		result.Skills[i].Action = "installed"
	}
	return result, nil
}

func resolveDestination(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		home, err := userHomeDirectory()
		if err != nil {
			return "", fmt.Errorf("resolve user home directory: %w", err)
		}
		if strings.TrimSpace(home) == "" {
			return "", errors.New("resolve user home directory: path is empty")
		}
		value = filepath.Join(home, ".agents", "skills")
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve skill destination: %w", err)
	}
	if info, statErr := os.Lstat(abs); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("skill destination must not be a symlink: %s", abs)
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return "", fmt.Errorf("inspect skill destination: %w", statErr)
	}
	return abs, nil
}

func findConflicts(destination string, pins []skillPin) ([]string, error) {
	info, err := os.Lstat(destination)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect skill destination: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("skill destination must be a real directory: %s", destination)
	}
	root, err := os.OpenRoot(destination)
	if err != nil {
		return nil, fmt.Errorf("open skill destination: %w", err)
	}
	defer func() { _ = root.Close() }()
	var conflicts []string
	for _, pin := range pins {
		entry, statErr := root.Lstat(pin.Name)
		if errors.Is(statErr, fs.ErrNotExist) {
			continue
		}
		if statErr != nil {
			return nil, fmt.Errorf("inspect existing skill %q: %w", pin.Name, statErr)
		}
		if entry.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("existing skill %q is a symlink", pin.Name)
		}
		conflicts = append(conflicts, pin.Name)
	}
	return conflicts, nil
}

func installationResult(destination string, pins []skillPin, conflicts []string, opts installOptions) skillInstallResult {
	conflictSet := make(map[string]bool, len(conflicts))
	for _, name := range conflicts {
		conflictSet[name] = true
	}
	items := make([]skillInstallItem, 0, len(pins))
	for _, pin := range pins {
		action := "install"
		if conflictSet[pin.Name] {
			if opts.Force {
				action = "replace"
			} else {
				action = "conflict"
			}
		}
		items = append(items, skillInstallItem{Name: pin.Name, TreeHash: pin.TreeHash, Action: action})
	}
	return skillInstallResult{
		Version:          1,
		Mode:             "preview",
		SourceRepository: skillsSourceRepositoryURL,
		SourceCommit:     skillsSourceCommit,
		Destination:      destination,
		Force:            opts.Force,
		Skills:           items,
	}
}

func defaultCheckoutPinnedSkills(ctx context.Context, workspace string) (string, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return "", fmt.Errorf("git is required to fetch reviewed skills commit %s: %w", skillsSourceCommit, err)
	}
	gitPath, err = filepath.Abs(gitPath)
	if err != nil {
		return "", fmt.Errorf("resolve git executable: %w", err)
	}
	source := filepath.Join(workspace, "source")
	home := filepath.Join(workspace, "git-home")
	hooks := filepath.Join(workspace, "git-hooks")
	if err := os.MkdirAll(filepath.Join(home, ".config"), 0o700); err != nil {
		return "", fmt.Errorf("create isolated git home: %w", err)
	}
	if err := os.MkdirAll(hooks, 0o700); err != nil {
		return "", fmt.Errorf("create isolated git hooks directory: %w", err)
	}
	env := isolatedGitEnvironment(home)
	run := func(args ...string) (string, error) {
		command := exec.CommandContext(ctx, gitPath, args...)
		command.Dir = workspace
		command.Env = env
		output, runErr := command.CombinedOutput()
		if runErr != nil {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), runErr, strings.TrimSpace(string(output)))
		}
		return strings.TrimSpace(string(output)), nil
	}
	config := []string{
		"-c", "core.hooksPath=" + hooks,
		"-c", "core.autocrlf=false",
		"-c", "protocol.file.allow=never",
		"-c", "protocol.ext.allow=never",
		"-c", "protocol.ssh.allow=never",
	}
	if _, err := run(append(config, "init", "--quiet", source)...); err != nil {
		return "", err
	}
	within := append(append([]string(nil), config...), "-C", source)
	if _, err := run(append(within, "fetch", "--quiet", "--depth=1", "--no-tags", skillsSourceRepositoryURL, skillsSourceCommit)...); err != nil {
		return "", err
	}
	if _, err := run(append(within, "checkout", "--quiet", "--detach", "FETCH_HEAD")...); err != nil {
		return "", err
	}
	commit, err := run(append(within, "rev-parse", "HEAD")...)
	if err != nil {
		return "", err
	}
	if commit != skillsSourceCommit {
		return "", fmt.Errorf("checked out commit %s, want %s", commit, skillsSourceCommit)
	}
	return source, nil
}

func isolatedGitEnvironment(home string) []string {
	filtered := make([]string, 0, len(os.Environ())+6)
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		switch name {
		case "HOME", "XDG_CONFIG_HOME", "GIT_CONFIG_GLOBAL", "GIT_CONFIG_NOSYSTEM", "GIT_TERMINAL_PROMPT", "GCM_INTERACTIVE":
			continue
		}
		filtered = append(filtered, entry)
	}
	return append(
		filtered,
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"GIT_CONFIG_GLOBAL="+filepath.Join(home, "gitconfig"),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
		"GCM_INTERACTIVE=never",
	)
}

func validatePinnedPack(source string, pins []skillPin) error {
	skillsDir := filepath.Join(source, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return fmt.Errorf("read pinned skill pack: %w", err)
	}
	want := make(map[string]string, len(pins))
	for _, pin := range pins {
		want[pin.Name] = pin.TreeHash
	}
	if len(entries) != len(want) {
		return fmt.Errorf("pinned checkout contains %d skills, want exactly %d", len(entries), len(want))
	}
	for _, entry := range entries {
		expectedHash, ok := want[entry.Name()]
		if !ok {
			return fmt.Errorf("pinned checkout contains unexpected skill %q", entry.Name())
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("inspect pinned skill %q: %w", entry.Name(), err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("pinned skill %q is not a real directory", entry.Name())
		}
		skillDir := filepath.Join(skillsDir, entry.Name())
		skillInfo, statErr := os.Lstat(filepath.Join(skillDir, "SKILL.md"))
		if statErr != nil || !skillInfo.Mode().IsRegular() {
			return fmt.Errorf("pinned skill %q does not contain a regular SKILL.md", entry.Name())
		}
		actualHash, err := gitTreeHash(skillDir)
		if err != nil {
			return fmt.Errorf("hash pinned skill %q: %w", entry.Name(), err)
		}
		if actualHash != expectedHash {
			return fmt.Errorf("pinned skill %q tree hash %s does not match reviewed hash %s", entry.Name(), actualHash, expectedHash)
		}
	}
	return nil
}

type gitTreeEntry struct {
	name string
	mode string
	hash [sha1.Size]byte
	dir  bool
}

func gitTreeHash(directory string) (string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", err
	}
	objects := make([]gitTreeEntry, 0, len(entries))
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing symlink %s", path)
		}
		object := gitTreeEntry{name: entry.Name()}
		switch {
		case info.IsDir():
			child, err := gitTreeHash(path)
			if err != nil {
				return "", err
			}
			raw, err := hex.DecodeString(child)
			if err != nil {
				return "", err
			}
			copy(object.hash[:], raw)
			object.mode = "40000"
			object.dir = true
		case info.Mode().IsRegular():
			data, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			object.hash = gitObjectHash("blob", data)
			object.mode = "100644"
			if info.Mode().Perm()&0o111 != 0 {
				object.mode = "100755"
			}
		default:
			return "", fmt.Errorf("refusing special file %s", path)
		}
		objects = append(objects, object)
	}
	sort.Slice(objects, func(i, j int) bool {
		left, right := objects[i].name, objects[j].name
		if objects[i].dir {
			left += "/"
		}
		if objects[j].dir {
			right += "/"
		}
		return left < right
	})
	var body strings.Builder
	for _, object := range objects {
		body.WriteString(object.mode)
		body.WriteByte(' ')
		body.WriteString(object.name)
		body.WriteByte(0)
		body.Write(object.hash[:])
	}
	hash := gitObjectHash("tree", []byte(body.String()))
	return hex.EncodeToString(hash[:]), nil
}

func gitObjectHash(kind string, content []byte) [sha1.Size]byte {
	header := fmt.Sprintf("%s %d%c", kind, len(content), byte(0))
	hasher := sha1.New() // #nosec G401 -- compatibility with reviewed Git object IDs.
	_, _ = io.WriteString(hasher, header)
	_, _ = hasher.Write(content)
	var result [sha1.Size]byte
	copy(result[:], hasher.Sum(nil))
	return result
}

func installPack(source string, destination *os.Root, pins []skillPin, force bool) (resultErr error) {
	sourceRoot, err := os.OpenRoot(filepath.Join(source, "skills"))
	if err != nil {
		return fmt.Errorf("open verified skill source: %w", err)
	}
	defer func() { _ = sourceRoot.Close() }()
	suffix, err := randomSuffix()
	if err != nil {
		return err
	}
	stageName := ".gplay-stage-" + suffix
	backupName := ".gplay-backup-" + suffix
	if err := destination.Mkdir(stageName, 0o700); err != nil {
		return fmt.Errorf("create install stage: %w", err)
	}
	stagePresent := true
	backupPresent := false
	defer func() {
		if stagePresent {
			_ = destination.RemoveAll(stageName)
		}
		if resultErr == nil && backupPresent {
			_ = destination.RemoveAll(backupName)
		}
	}()
	for _, pin := range pins {
		if err := copyRootTree(sourceRoot, destination, pin.Name, filepath.Join(stageName, pin.Name)); err != nil {
			return fmt.Errorf("stage skill %q: %w", pin.Name, err)
		}
	}

	var movedExisting []string
	if force {
		if err := destination.Mkdir(backupName, 0o700); err != nil {
			return fmt.Errorf("create skill backup: %w", err)
		}
		backupPresent = true
		for _, pin := range pins {
			if _, statErr := destination.Lstat(pin.Name); errors.Is(statErr, fs.ErrNotExist) {
				continue
			} else if statErr != nil {
				return fmt.Errorf("inspect existing skill %q: %w", pin.Name, statErr)
			}
			if err := renameWithinRoot(destination, pin.Name, filepath.Join(backupName, pin.Name)); err != nil {
				return rollbackPack(destination, nil, movedExisting, backupName, fmt.Errorf("back up skill %q: %w", pin.Name, err))
			}
			movedExisting = append(movedExisting, pin.Name)
		}
	}

	var installed []string
	for _, pin := range pins {
		if err := renameWithinRoot(destination, filepath.Join(stageName, pin.Name), pin.Name); err != nil {
			return rollbackPack(destination, installed, movedExisting, backupName, fmt.Errorf("install skill %q: %w", pin.Name, err))
		}
		installed = append(installed, pin.Name)
	}
	if err := destination.RemoveAll(stageName); err != nil {
		return fmt.Errorf("remove install stage: %w", err)
	}
	stagePresent = false
	if backupPresent {
		if err := destination.RemoveAll(backupName); err != nil {
			return fmt.Errorf("remove completed skill backup: %w", err)
		}
		backupPresent = false
	}
	return nil
}

func rollbackPack(root *os.Root, installed, movedExisting []string, backupName string, installErr error) error {
	var rollbackErr error
	for i := len(installed) - 1; i >= 0; i-- {
		if err := root.RemoveAll(installed[i]); err != nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("remove partially installed %q: %w", installed[i], err))
		}
	}
	for i := len(movedExisting) - 1; i >= 0; i-- {
		name := movedExisting[i]
		if err := renameWithinRoot(root, filepath.Join(backupName, name), name); err != nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("restore previous skill %q: %w", name, err))
		}
	}
	if rollbackErr != nil {
		return errors.Join(installErr, fmt.Errorf("rollback failed; previous skill data remains under %s: %w", backupName, rollbackErr))
	}
	if err := root.RemoveAll(backupName); err != nil {
		return errors.Join(installErr, fmt.Errorf("remove restored skill backup %s: %w", backupName, err))
	}
	return installErr
}

func copyRootTree(source, destination *os.Root, sourceName, destinationName string) error {
	info, err := source.Lstat(sourceName)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing source symlink %q", sourceName)
	}
	if info.IsDir() {
		if err := destination.Mkdir(destinationName, 0o755); err != nil {
			return err
		}
		directory, err := source.Open(sourceName)
		if err != nil {
			return err
		}
		entries, readErr := directory.ReadDir(-1)
		_ = directory.Close()
		if readErr != nil {
			return readErr
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			if err := copyRootTree(source, destination, filepath.Join(sourceName, entry.Name()), filepath.Join(destinationName, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing source special file %q", sourceName)
	}
	input, err := source.Open(sourceName)
	if err != nil {
		return err
	}
	defer input.Close()
	mode := info.Mode().Perm() & 0o755
	if mode == 0 {
		mode = 0o644
	}
	output, err := destination.OpenFile(destinationName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	syncErr := output.Sync()
	closeErr := output.Close()
	return errors.Join(copyErr, syncErr, closeErr)
}

func randomSuffix() (string, error) {
	var data [8]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", fmt.Errorf("generate install transaction ID: %w", err)
	}
	return hex.EncodeToString(data[:]), nil
}
