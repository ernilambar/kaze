package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"kaze/internal/pathutil"
)

// Git is read-only: the app inspects a repository's state so the sidebar, the
// gutter and the status bar can describe it, but every command it runs either
// prints status or a diff. --no-optional-locks keeps git from refreshing the
// index behind the user's back, which is the one write `git status` would
// otherwise make.

// GitRepo is one repository behind a sidebar root. Branch is the checked-out
// branch, or the short commit hash when HEAD is detached.
type GitRepo struct {
	Root   string `json:"root"`
	Branch string `json:"branch"`
	Ahead  int    `json:"ahead"`
	Behind int    `json:"behind"`
}

// GitChange is one file git reports as different from HEAD. Status is one of
// modified, added, deleted, renamed, copied, conflicted or untracked.
type GitChange struct {
	Path   string `json:"path"`
	Root   string `json:"root"`
	Status string `json:"status"`
}

// GitSnapshot is the whole read-only picture for the sidebar roots: the
// repositories they live in and the changed files inside them.
type GitSnapshot struct {
	Repos   []GitRepo   `json:"repos"`
	Changes []GitChange `json:"changes"`
}

// GitBaseline is the HEAD version of a file. The gutter diffs the buffer
// against it, so changed lines show while the user is still typing.
type GitBaseline struct {
	// InRepo says whether the path belongs to a repository at all. Untracked
	// files return InRepo=true and Tracked=false, because every line is new.
	InRepo  bool `json:"in_repo"`
	Tracked bool `json:"tracked"`
	// Ignored says the repository's ignore rules exclude the path, so it is
	// outside git's picture and the gutter shows no marks for it.
	Ignored bool   `json:"ignored"`
	Content string `json:"content"`
}

// HasHead reports whether the repository has a commit yet.
func HasHead(root string) bool {
	_, err := Output(root, "rev-parse", "--verify", "HEAD")
	return err == nil
}

// Tracked reports whether HEAD or the index knows the path.
func Tracked(root, relative string) bool {
	_, err := Output(root, "ls-files", "--error-unmatch", "--", gitPathspec(relative))
	return err == nil
}

// Ignored reports whether the ignore rules exclude the path. check-ignore
// never reports a tracked path, so a force-added file keeps its marks.
func Ignored(root, relative string) bool {
	_, err := Output(root, "check-ignore", "--quiet", "--", relative)
	return err == nil
}

// StagedDiff shows the index against the empty tree of an unborn branch.
func StagedDiff(root, relative string) ([]byte, error) {
	stdout, _, err := gitCommand(root,
		"diff", "--cached", "--no-color", "--no-ext-diff", "--no-textconv", "--unified=3", "--", gitPathspec(relative))
	return stdout, err
}

// ReadStatus runs porcelain v2 with NUL terminators and parses it. The
// machine-readable format carries branch, ahead/behind and per-file states in
// one command, so one process answers every sidebar and status-bar question.
func ReadStatus(root string) (GitRepo, []GitChange, error) {
	output, err := Output(root, "status", "--porcelain=v2", "-z", "--branch", "--untracked-files=all")
	if err != nil {
		return GitRepo{}, nil, err
	}
	return parseGitStatus(root, output)
}

// parseGitStatus decodes the NUL-separated records of porcelain v2. It is split
// from the command so the format can be tested without a repository on disk.
func parseGitStatus(root string, data []byte) (GitRepo, []GitChange, error) {
	repo := GitRepo{Root: root}
	changes := make([]GitChange, 0)

	tokens := bytes.Split(data, []byte{0})
	oid := ""
	// Rename and copy records carry the original path in a following token.
	skipNext := false
	for _, token := range tokens {
		if skipNext {
			skipNext = false
			continue
		}
		if len(token) == 0 {
			continue
		}
		line := string(token)

		switch {
		case strings.HasPrefix(line, "# branch.head "):
			repo.Branch = strings.TrimPrefix(line, "# branch.head ")
		case strings.HasPrefix(line, "# branch.oid "):
			oid = strings.TrimPrefix(line, "# branch.oid ")
		case strings.HasPrefix(line, "# branch.ab "):
			repo.Ahead, repo.Behind = parseAheadBehind(strings.TrimPrefix(line, "# branch.ab "))
		case strings.HasPrefix(line, "1 "):
			fields := strings.SplitN(line, " ", 9)
			if len(fields) == 9 {
				changes = append(changes, GitChange{
					Path:   filepath.Join(root, filepath.FromSlash(fields[8])),
					Root:   root,
					Status: gitStatusKind(fields[1]),
				})
			}
		case strings.HasPrefix(line, "2 "):
			fields := strings.SplitN(line, " ", 10)
			if len(fields) == 10 {
				changes = append(changes, GitChange{
					Path:   filepath.Join(root, filepath.FromSlash(fields[9])),
					Root:   root,
					Status: gitStatusKind(fields[1]),
				})
				skipNext = true
			}
		case strings.HasPrefix(line, "u "):
			fields := strings.SplitN(line, " ", 11)
			if len(fields) == 11 {
				changes = append(changes, GitChange{
					Path:   filepath.Join(root, filepath.FromSlash(fields[10])),
					Root:   root,
					Status: "conflicted",
				})
			}
		case strings.HasPrefix(line, "? "):
			changes = append(changes, GitChange{
				Path:   filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(line, "? "))),
				Root:   root,
				Status: "untracked",
			})
		}
	}

	// A detached HEAD has no branch name of its own, so the short commit hash
	// identifies the checkout in the status bar.
	if repo.Branch == "(detached)" && len(oid) >= 7 {
		repo.Branch = oid[:7]
	}
	return repo, changes, nil
}

// parseAheadBehind reads the "+N -M" counts of the branch.ab header. A branch
// without an upstream has no header, which leaves both counts at zero.
func parseAheadBehind(value string) (ahead, behind int) {
	fields := strings.Fields(value)
	for _, field := range fields {
		if len(field) < 2 {
			continue
		}
		count, err := strconv.Atoi(field[1:])
		if err != nil {
			continue
		}
		switch field[0] {
		case '+':
			ahead = count
		case '-':
			behind = count
		}
	}
	return ahead, behind
}

// gitStatusKind maps porcelain v2's two-letter XY state to the vocabulary the
// sidebar uses. The worktree letter wins when both sides changed: it describes
// what the user sees in the file right now.
func gitStatusKind(xy string) string {
	if len(xy) != 2 {
		return "modified"
	}
	if strings.Contains(xy, "U") || xy == "AA" || xy == "DD" {
		return "conflicted"
	}
	letter := xy[1]
	if letter == '.' {
		letter = xy[0]
	}
	switch letter {
	case 'A':
		return "added"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	case 'C':
		return "copied"
	default:
		return "modified"
	}
}

// RepoRoot returns the top level of the repository containing dir, or an
// empty string when git is missing or dir is not in a repository.
func RepoRoot(dir string) string {
	output, err := Output(pathutil.CanonicalPath(dir), "rev-parse", "--show-toplevel")
	if err != nil {
		return ""
	}
	return pathutil.CanonicalPath(strings.TrimSpace(string(output)))
}

// RepoRootForFile resolves the repository for a file path, including a
// deleted file whose own directory may still exist.
func RepoRootForFile(path string) string {
	return RepoRoot(filepath.Dir(path))
}

// RelativePath is the repository-relative, slash-separated name git expects
// on the command line. Git always speaks forward slashes.
func RelativePath(root, path string) (string, error) {
	relative, err := filepath.Rel(root, pathutil.CanonicalPath(path))
	if err != nil || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s is outside %s", path, root)
	}
	return filepath.ToSlash(relative), nil
}

// gitPathspec names one file literally, so a name containing *, ? or [ is not
// read as a pattern.
func gitPathspec(relative string) string {
	return ":(literal)" + relative
}

// SidebarPath expresses a physical repository path under the lexical root the
// user added to the sidebar. The returned root is that same lexical folder, so
// the changes list can show a path relative to it. The frontend builds tree
// entries from that root, so status marks only match when the prefix agrees.
func SidebarPath(path string, folders []string) (string, string) {
	for _, folder := range folders {
		relative, err := filepath.Rel(pathutil.CanonicalPath(folder), path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		return filepath.Join(folder, relative), folder
	}
	return path, ""
}

// Output runs one read-only git command in dir.
func Output(dir string, args ...string) ([]byte, error) {
	stdout, _, err := gitCommand(dir, args...)
	return stdout, err
}

// DiffOutput runs `git diff HEAD` for one file.
func DiffOutput(root, relative string) ([]byte, error) {
	stdout, _, err := gitCommand(root,
		"diff", "HEAD", "--no-color", "--no-ext-diff", "--no-textconv", "--unified=3", "--", gitPathspec(relative))
	return stdout, err
}

// UntrackedDiff shows a file HEAD does not contain as entirely added.
// `git diff --no-index` exits 1 when the inputs differ, which is the normal
// case here.
func UntrackedDiff(root, relative string) (string, error) {
	stdout, _, err := gitCommandAllowingDifferences(root,
		"diff", "--no-index", "--no-color", "--no-ext-diff", "--no-textconv", "--unified=3", "--", "/dev/null", relative)
	if err != nil {
		return "", err
	}
	return string(stdout), nil
}

// gitCommand runs git with the flags that keep it read-only. The environment
// stops git from prompting for credentials in a GUI process that has no
// terminal to answer on.
func gitCommand(dir string, args ...string) (stdout []byte, stderr []byte, err error) {
	return runGit(false, dir, args...)
}

func gitCommandAllowingDifferences(dir string, args ...string) (stdout []byte, stderr []byte, err error) {
	return runGit(true, dir, args...)
}

// A repository's own config is untrusted: a cloned or downloaded folder can
// name a program in core.fsmonitor that every `git status` would run. Kaze only
// reads, so the hook is switched off for every call.
func runGit(allowExitOne bool, dir string, args ...string) (stdout []byte, stderr []byte, err error) {
	prefix := []string{"--no-optional-locks", "-c", "core.fsmonitor=false", "-C", dir}
	command := exec.Command("git", append(prefix, args...)...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0")
	var out, errOut bytes.Buffer
	command.Stdout = &out
	command.Stderr = &errOut

	err = command.Run()
	if err != nil {
		var exitError *exec.ExitError
		if allowExitOne && errors.As(err, &exitError) && exitError.ExitCode() == 1 {
			return out.Bytes(), errOut.Bytes(), nil
		}
		if errors.Is(err, exec.ErrNotFound) {
			return nil, errOut.Bytes(), fmt.Errorf("git is not installed")
		}
		message := strings.TrimSpace(errOut.String())
		if message == "" {
			message = err.Error()
		}
		return nil, errOut.Bytes(), fmt.Errorf("%s", message)
	}
	return out.Bytes(), errOut.Bytes(), nil
}
