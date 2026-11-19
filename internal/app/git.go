package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"kaze/internal/document"
	"kaze/internal/git"
	"kaze/internal/pathutil"
)

// GetGitStatus reports the repositories that own the sidebar roots and the
// changed files inside them. paths are one window's open files and expanded
// folders: they let a repository nested under a root contribute its status
// without walking the whole tree. A root outside a repository is skipped
// silently: a non-repository folder is normal, not a failure.
func (a *App) GetGitStatus(folders []string, paths []string) (git.GitSnapshot, error) {
	snapshot := git.GitSnapshot{Repos: []git.GitRepo{}, Changes: []git.GitChange{}}
	seenRoots := make(map[string]bool, len(folders)+len(paths))
	seenChanges := make(map[string]bool)

	// Repositories are collected in a fixed order: the roots first, then the
	// ones the watched paths reveal, so the branch and the changes list stay
	// stable between reads.
	roots := make([]string, 0, len(folders)+len(paths))
	addRoot := func(root string) {
		if root == "" || seenRoots[root] {
			return
		}
		seenRoots[root] = true
		roots = append(roots, root)
	}
	for _, folder := range folders {
		addRoot(git.RepoRoot(folder))
	}
	for _, path := range paths {
		// Git reports physical paths, so compare against the canonical one:
		// a sidebar rooted at /var and a repository at /private/var are the
		// same tree. A tab can point outside the sidebar; its repository
		// cannot describe anything the sidebar shows.
		canonical := pathutil.CanonicalPath(path)
		if !pathutil.WithinAny(canonical, folders) {
			continue
		}
		// A path already inside a known repository needs no git process.
		known := false
		for root := range seenRoots {
			if pathutil.IsWithin(root, canonical) {
				known = true
				break
			}
		}
		if !known {
			addRoot(repoRootForPath(path))
		}
	}

	for _, root := range roots {
		repo, changes, err := git.ReadStatus(root)
		if err != nil {
			continue
		}
		// The frontend knows only the roots the user added. Naming the
		// repository by one of those keeps branch selection working when the
		// folder path goes through a symlink.
		repo.Root = gitDisplayRoot(root, folders)
		snapshot.Repos = append(snapshot.Repos, repo)
		for _, change := range changes {
			if !pathutil.WithinAny(change.Path, folders) || seenChanges[change.Path] {
				continue
			}
			seenChanges[change.Path] = true
			// Git reports the repository's physical path. The sidebar uses
			// whatever path the user added, so the change is expressed under
			// that root again; on macOS /var and /private/var are one folder.
			change.Path, _ = git.SidebarPath(change.Path, folders)
			change.Root = repo.Root
			snapshot.Changes = append(snapshot.Changes, change)
		}
	}

	sort.Slice(snapshot.Changes, func(i, j int) bool {
		return snapshot.Changes[i].Path < snapshot.Changes[j].Path
	})
	return snapshot, nil
}

// repoRootForPath resolves the repository owning path, whether path is a file
// or a folder. A path that no longer exists resolves through its parent, the
// same way a deleted tab still knows where it lived.
func repoRootForPath(path string) string {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return git.RepoRoot(path)
	}
	return git.RepoRootForFile(path)
}

// gitDisplayRoot names a repository the way the sidebar spells it. A repository
// nested under a root takes the root's spelling; one wider than the root keeps
// the folder the user added, so the status bar still finds it by prefix.
func gitDisplayRoot(root string, folders []string) string {
	if display, _ := git.SidebarPath(root, folders); pathutil.WithinAny(pathutil.CanonicalPath(display), folders) {
		return display
	}
	for _, folder := range folders {
		if pathutil.IsWithin(root, pathutil.CanonicalPath(folder)) {
			return folder
		}
	}
	return root
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
