import { describe, expect, it } from 'vitest';
import {
    fuzzyScore,
    rankQuickOpen,
    toQuickOpenFile,
    toQuickOpenFiles,
    type QuickOpenFile,
} from './quickOpen';

function file(name: string, detail = 'src'): QuickOpenFile {
    return { path: `/${detail}/${name}`, name, detail };
}

describe('toQuickOpenFiles', () => {
    it('labels a file with its root and parent folders', () => {
        expect(toQuickOpenFiles(['/repo/frontend/src/App.svelte'], ['/repo'])).toEqual([
            { path: '/repo/frontend/src/App.svelte', name: 'App.svelte', detail: 'repo/frontend/src' },
        ]);
    });

    it('tolerates a trailing separator on the root', () => {
        const [entry] = toQuickOpenFiles(['/repo/notes.txt'], ['/repo/']);

        expect(entry?.detail).toBe('repo');
    });

    it('uses the innermost root when roots overlap', () => {
        const entry = toQuickOpenFile('/repo/frontend/src/App.svelte', [
            '/repo',
            '/repo/frontend',
        ]);

        expect(entry.detail).toBe('frontend/src');
    });

    it('falls back to the absolute parent when no root contains the path', () => {
        const entry = toQuickOpenFile('/elsewhere/notes.txt', ['/repo']);

        expect(entry).toEqual({ path: '/elsewhere/notes.txt', name: 'notes.txt', detail: '/elsewhere' });
    });
});

describe('fuzzyScore', () => {
    it('matches an in-order subsequence', () => {
        expect(fuzzyScore('apsv', 'App.svelte')).toBeDefined();
    });

    it('ignores case', () => {
        expect(fuzzyScore('APPS', 'App.svelte')).toBeDefined();
    });

    it('rejects out-of-order or missing characters', () => {
        expect(fuzzyScore('svap', 'App.svelte')).toBeUndefined();
        expect(fuzzyScore('zzz', 'App.svelte')).toBeUndefined();
    });

    it('rewards matches at a word boundary', () => {
        const start = fuzzyScore('app', 'app.ts');
        const inside = fuzzyScore('app', 'xapp.ts');

        expect(start).toBeDefined();
        expect(inside).toBeDefined();
        expect(start ?? 0).toBeGreaterThan(inside ?? 0);
    });
});

describe('rankQuickOpen', () => {
    it('returns the first files for an empty query', () => {
        const files = [file('a.ts'), file('b.ts'), file('c.ts')];

        expect(rankQuickOpen('', files, 2)).toEqual(files.slice(0, 2));
    });

    it('matches by file name', () => {
        const files = [file('App.svelte', 'frontend'), file('main.go', 'cmd')];

        expect(rankQuickOpen('appsv', files)[0]?.name).toBe('App.svelte');
    });

    it('matches by path', () => {
        const files = [file('App.svelte', 'frontend/src'), file('app.ts', 'backend')];

        expect(rankQuickOpen('frontend/src/app', files)[0]?.name).toBe('App.svelte');
    });

    it('prefers a name match over a folder match', () => {
        const files = [file('notes.txt', 'app'), file('app.ts', 'docs')];

        expect(rankQuickOpen('app', files)[0]?.name).toBe('app.ts');
    });

    it('ranks an exact name above a longer prefix', () => {
        const files = [file('app.tsx', 'a'), file('app.ts', 'z')];

        expect(rankQuickOpen('app.ts', files)[0]?.name).toBe('app.ts');
    });

    it('respects the limit', () => {
        const files = ['a.ts', 'b.ts', 'c.ts'].map((name) => file(name));

        expect(rankQuickOpen('ts', files, 2)).toHaveLength(2);
    });

    it('excludes files that do not match', () => {
        expect(rankQuickOpen('zzz', [file('app.ts')])).toEqual([]);
    });
});
