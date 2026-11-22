import { describe, expect, it } from 'vitest';
import {
    branchLabel,
    changePath,
    changedFolders,
    repoForPath,
    statusLabel,
    statusMark,
    type GitChange,
    type GitRepo,
} from './git';

function repo(root: string, branch = 'main', ahead = 0, behind = 0): GitRepo {
    return { root, branch, ahead, behind };
}

describe('branchLabel', () => {
    it('is empty without a repository', () => {
        expect(branchLabel(undefined)).toBe('');
    });

    it('prints only the branch when it matches its upstream', () => {
        expect(branchLabel(repo('/repo'))).toBe('main');
    });

    it('adds ahead and behind counts', () => {
        expect(branchLabel(repo('/repo', 'dev', 2, 3))).toBe('dev ↑2 ↓3');
    });

    it('keeps the short hash of a detached HEAD', () => {
        expect(branchLabel(repo('/repo', 'abc1234'))).toBe('abc1234');
    });
});

describe('repoForPath', () => {
    const repos = [repo('/work'), repo('/work/nested')];

    it('picks the innermost repository containing the file', () => {
        expect(repoForPath(repos, '/work/nested/src/file.ts')?.root).toBe('/work/nested');
    });

    it('falls back to the outermost repository for a file outside both', () => {
        expect(repoForPath(repos, '/elsewhere/file.ts')?.root).toBe('/work');
    });

    it('uses the first repository when no file is open', () => {
        expect(repoForPath(repos, '')?.root).toBe('/work');
    });

    it('is undefined when there are no repositories', () => {
        expect(repoForPath([], '/work/file.ts')).toBeUndefined();
    });
});

describe('status marks', () => {
    it('uses one letter per state', () => {
        expect(statusMark('modified')).toBe('M');
        expect(statusMark('added')).toBe('A');
        expect(statusMark('deleted')).toBe('D');
        expect(statusMark('untracked')).toBe('U');
        expect(statusMark('renamed')).toBe('R');
        expect(statusMark('copied')).toBe('C');
        expect(statusMark('conflicted')).toBe('!');
    });

    it('spells the state for the tooltip', () => {
        expect(statusLabel('untracked')).toBe('Untracked');
        expect(statusLabel('modified')).toBe('Modified');
    });
});

describe('changePath', () => {
    const change: GitChange = {
        path: '/work/src/file.ts',
        root: '/work',
        status: 'modified',
    };

    it('is relative to the repository root', () => {
        expect(changePath(change)).toBe('src/file.ts');
    });
});

describe('changedFolders', () => {
    function change(path: string, status: GitChange['status'] = 'modified'): GitChange {
        return { path, root: '/work', status };
    }

    it('marks a change on its own path and every folder above it', () => {
        const folders = changedFolders([change('/work/src/app/file.ts')], ['/work']);
        expect(folders.has('/work/src/app/file.ts')).toBe(true);
        expect(folders.has('/work/src/app')).toBe(true);
        expect(folders.has('/work/src')).toBe(true);
        expect(folders.has('/work')).toBe(true);
    });

    it('stops at the sidebar root', () => {
        const folders = changedFolders([change('/work/src/file.ts')], ['/work']);
        expect(folders.has('/')).toBe(false);
        expect(folders.size).toBe(3);
    });

    it('does not mark a folder for a deleted file', () => {
        const folders = changedFolders(
            [change('/work/src/deleted.ts', 'deleted')],
            ['/work'],
        );
        expect(folders.size).toBe(0);
    });

    it('lets a visible change win over a deletion in the same folder', () => {
        const folders = changedFolders(
            [
                change('/work/src/deleted.ts', 'deleted'),
                change('/work/src/modified.ts', 'modified'),
            ],
            ['/work'],
        );
        expect(folders.get('/work/src')).toBe('modified');
        expect(folders.get('/work')).toBe('modified');
    });

    it('marks a folder once however many changes it holds', () => {
        const folders = changedFolders(
            [change('/work/src/a.ts'), change('/work/src/b.ts', 'untracked')],
            ['/work'],
        );
        expect(folders.get('/work/src')).toBe('modified');
        expect(folders.size).toBe(4);
    });

    it('keeps the most significant status for a folder', () => {
        const folders = changedFolders(
            [
                change('/work/src/a.ts', 'added'),
                change('/work/src/b.ts', 'modified'),
                change('/work/src/c.ts', 'conflicted'),
            ],
            ['/work'],
        );
        expect(folders.get('/work/src')).toBe('conflicted');
        expect(folders.get('/work')).toBe('conflicted');
    });

    it('leaves paths outside every root out', () => {
        const folders = changedFolders([change('/elsewhere/file.ts')], ['/work']);
        expect(folders.size).toBe(0);
    });
});
