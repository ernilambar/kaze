package app

import (
	"fmt"
	"path/filepath"
	"sort"

	"kaze/internal/config"
	"kaze/internal/document"
	"kaze/internal/git"
	"kaze/internal/ignore"
	"kaze/internal/pathutil"
)

// GetGitStatus reports the repositories that own the sidebar roots and the
// changed files inside them. A root outside a repository is a container of
// repositories; discover walks it for the ones it holds, so the changes list
// does not depend on which folders the tree has expanded. A folder with no
// repository stays absent silently: that is normal, not a failure.
func (a *App) GetGitStatus(folders []string) (git.GitSnapshot, error) {
	snapshot := git.GitSnapshot{Repos: []git.GitRepo{}, Changes: []git.GitChange{}}
	seenRoots := make(map[string]bool, len(folders))
	seenChanges := make(map[string]bool)

	// Repositories are collected in a fixed order, so the branch and the
	// changes list stay stable between reads.
	roots := make([]string, 0, len(folders))
	addRoot := func(root string) {
		if root == "" {
			return
		}
		// Git reports physical paths, so the reads and the change filter must
		// use the canonical spelling even when the sidebar root goes through a
		// symlink.
		root = pathutil.CanonicalPath(root)
		if seenRoots[root] {
			return
		}
		seenRoots[root] = true
		roots = append(roots, root)
	}
	excludes := fileExcludes()
	for _, folder := range folders {
		// A root inside a repository is covered by that repository, including
		// changes below the root. Only a plain container needs a walk.
		if root := git.RepoRoot(folder); root != "" {
			addRoot(root)
			continue
		}
		for _, nested := range git.DiscoverRepos(folder, excludes) {
			addRoot(nested)
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

// fileExcludes reads files.exclude the way the tree does, so a directory the
// explorer hides is not walked for repositories. A missing or broken settings
// file hides nothing: git status still works while the user repairs it.
func fileExcludes() ignore.Matcher {
	if path, err := settingsFilePath(); err == nil {
		if settings, err := config.Load(path); err == nil {
			return ignore.NewMatcher(settings.Files.Exclude)
		}
	}
	return ignore.Matcher{}
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
