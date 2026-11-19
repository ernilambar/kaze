package app

import (
	"fmt"
	"path/filepath"
	"sort"

	"kaze/internal/document"
	"kaze/internal/git"
	"kaze/internal/pathutil"
)

// GetGitStatus reports the repositories behind the sidebar roots and the
// changed files inside them. A root outside a repository is skipped silently:
// a non-repository folder is normal, not a failure.
func (a *App) GetGitStatus(folders []string) (git.GitSnapshot, error) {
	snapshot := git.GitSnapshot{Repos: []git.GitRepo{}, Changes: []git.GitChange{}}
	seenRoots := make(map[string]bool, len(folders))
	seenChanges := make(map[string]bool)

	for _, folder := range folders {
		root := git.RepoRoot(folder)
		if root == "" || seenRoots[root] {
			continue
		}
		seenRoots[root] = true

		repo, changes, err := git.ReadStatus(root)
		if err != nil {
			continue
		}
		// The frontend knows only the roots the user added. Naming the
		// repository by one of those keeps branch selection working when the
		// folder path goes through a symlink.
		repo.Root = folder
		snapshot.Repos = append(snapshot.Repos, repo)
		for _, change := range changes {
			if !pathutil.WithinAny(change.Path, folders) || seenChanges[change.Path] {
				continue
			}
			seenChanges[change.Path] = true
			// Git reports the repository's physical path. The sidebar uses
			// whatever path the user added, so the change is expressed under
			// that root again; on macOS /var and /private/var are one folder.
			change.Path, change.Root = git.SidebarPath(change.Path, folders)
			snapshot.Changes = append(snapshot.Changes, change)
		}
	}

	sort.Slice(snapshot.Changes, func(i, j int) bool {
		return snapshot.Changes[i].Path < snapshot.Changes[j].Path
	})
	return snapshot, nil
}

// GetGitBaseline returns the committed text of path. A path outside a
// repository, or one HEAD does not contain, reports that instead of an error:
// the gutter only needs to know whether it has a baseline to compare against.
func (a *App) GetGitBaseline(path string) (git.GitBaseline, error) {
	root := git.RepoRootForFile(path)
	if root == "" {
		return git.GitBaseline{}, nil
	}
	relative, err := git.RelativePath(root, path)
	if err != nil {
		return git.GitBaseline{}, nil
	}
	if _, err := git.Output(root, "rev-parse", "--verify", "HEAD"); err != nil {
		// An unborn branch has no HEAD, so an existing file is entirely new.
		// An ignored file stays outside git's picture and gets no marks.
		return git.GitBaseline{InRepo: true, Ignored: git.Ignored(root, relative)}, nil
	}

	content, err := git.Output(root, "show", "HEAD:"+relative)
	if err != nil {
		return git.GitBaseline{InRepo: true, Ignored: git.Ignored(root, relative)}, nil
	}
	if len(content) > document.MaxSize {
		return git.GitBaseline{}, fmt.Errorf("%s is too large to diff", filepath.Base(path))
	}
	return git.GitBaseline{InRepo: true, Tracked: true, Content: string(content)}, nil
}

// GetGitDiff returns a unified diff of path against HEAD. Tracked changes use
// `git diff HEAD`, which includes both staged and unstaged edits; an untracked
// file is shown as an added file instead.
func (a *App) GetGitDiff(path string) (string, error) {
	root := git.RepoRootForFile(path)
	if root == "" {
		return "", fmt.Errorf("%s is not in a git repository", filepath.Base(path))
	}
	relative, err := git.RelativePath(root, path)
	if err != nil {
		return "", err
	}

	tracked := git.Tracked(root, relative)
	var diff string
	if git.HasHead(root) {
		if tracked {
			output, err := git.DiffOutput(root, relative)
			if err != nil {
				return "", err
			}
			diff = string(output)
		} else {
			diff, err = git.UntrackedDiff(root, relative)
			if err != nil {
				return "", err
			}
		}
	} else if tracked {
		// An unborn branch has no HEAD to compare against, so staged content is
		// the closest thing to the committed side.
		output, err := git.StagedDiff(root, relative)
		if err != nil {
			return "", err
		}
		diff = string(output)
	} else {
		diff, err = git.UntrackedDiff(root, relative)
		if err != nil {
			return "", err
		}
	}

	if len(diff) > document.MaxSize {
		return "", fmt.Errorf("the diff is larger than %d MB", document.MaxSize>>20)
	}
	return diff, nil
}
