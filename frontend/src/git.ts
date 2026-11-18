import type * as models from '../bindings/kaze/internal/git/models';
import { GetGitBaseline, GetGitDiff, GetGitStatus } from '../bindings/kaze/app';
import { isWithin, relativePath } from './sidebar';

// The status vocabulary mirrors git.go. `modified` covers staged and unstaged
// edits; `untracked` files are unseen by HEAD entirely.
export type GitStatusKind =
    | 'modified'
    | 'added'
    | 'deleted'
    | 'untracked'
    | 'renamed'
    | 'copied'
    | 'conflicted';

export interface GitRepo {
    root: string;
    branch: string;
    ahead: number;
    behind: number;
}

export interface GitChange {
    path: string;
    root: string;
    status: GitStatusKind;
}

export interface GitState {
    repos: GitRepo[];
    changes: GitChange[];
}

export interface GitBaseline {
    inRepo: boolean;
    tracked: boolean;
    content: string;
}

// SidebarView is which list the sidebar shows. The status bar switches between
// the two, matching how a Git-aware editor exposes its changes.
export type SidebarView = 'explorer' | 'changes';

export const emptyGitState: GitState = { repos: [], changes: [] };

function toStatusKind(status: string): GitStatusKind {
    switch (status) {
        case 'added':
        case 'deleted':
        case 'untracked':
        case 'renamed':
        case 'copied':
        case 'conflicted':
            return status;
        default:
            return 'modified';
    }
}

function toRepo(repo: models.GitRepo): GitRepo {
    return {
        root: repo.root,
        branch: repo.branch,
        ahead: repo.ahead,
        behind: repo.behind,
    };
}

function toChange(change: models.GitChange): GitChange {
    return {
        path: change.path,
        root: change.root,
        status: toStatusKind(change.status),
    };
}

// loadGitStatus reads every sidebar root's repository state. A folder outside a
// repository is simply absent from the result, not an error.
export async function loadGitStatus(folders: readonly string[]): Promise<GitState> {
    const snapshot = await GetGitStatus(folders.slice());
    return {
        repos: (snapshot?.repos ?? []).map(toRepo),
        changes: (snapshot?.changes ?? []).map(toChange),
    };
}

export async function readGitBaseline(path: string): Promise<GitBaseline> {
    const baseline = await GetGitBaseline(path);
    return {
        inRepo: baseline?.in_repo ?? false,
        tracked: baseline?.tracked ?? false,
        content: baseline?.content ?? '',
    };
}

// readGitDiff returns the unified diff for one file. An empty string means the
// file matches HEAD, which the caller turns into a short explanation.
export async function readGitDiff(path: string): Promise<string> {
    return (await GetGitDiff(path)) ?? '';
}

// repoForPath picks the repository containing the active file, falling back to
// the first root so the branch is still useful when nothing is open.
export function repoForPath(repos: readonly GitRepo[], path: string): GitRepo | undefined {
    if (repos.length === 0) {
        return undefined;
    }
    if (path === '') {
        return repos[0];
    }
    let match: GitRepo | undefined;
    for (const repo of repos) {
        if (isWithin(repo.root, path) && (match === undefined || repo.root.length > match.root.length)) {
            match = repo;
        }
    }
    return match ?? repos[0];
}

// branchLabel is what the status bar prints: the branch plus the counts that
// matter when it has an upstream.
export function branchLabel(repo: GitRepo | undefined): string {
    if (repo === undefined || repo.branch === '') {
        return '';
    }
    let label = repo.branch;
    if (repo.ahead > 0) {
        label += ` ↑${repo.ahead}`;
    }
    if (repo.behind > 0) {
        label += ` ↓${repo.behind}`;
    }
    return label;
}

// statusMark is the one-letter mark the explorer and the changes list show.
export function statusMark(status: GitStatusKind): string {
    switch (status) {
        case 'added':
            return 'A';
        case 'deleted':
            return 'D';
        case 'untracked':
            return 'U';
        case 'renamed':
            return 'R';
        case 'copied':
            return 'C';
        case 'conflicted':
            return '!';
        default:
            return 'M';
    }
}

export function statusLabel(status: GitStatusKind): string {
    switch (status) {
        case 'added':
            return 'Added';
        case 'deleted':
            return 'Deleted';
        case 'untracked':
            return 'Untracked';
        case 'renamed':
            return 'Renamed';
        case 'copied':
            return 'Copied';
        case 'conflicted':
            return 'Conflicted';
        default:
            return 'Modified';
    }
}

// changePath is the path shown in the changes list: relative to its repository
// root, so a deep change stays readable.
export function changePath(change: GitChange): string {
    const relative = relativePath(change.path, [change.root]);
    return relative === '' ? change.path : relative;
}
