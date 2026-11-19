import { describe, expect, it } from 'vitest';
import { changedLines, diffLineKinds } from './diff';

describe('changedLines', () => {
    it('reports nothing for identical text', () => {
        expect(changedLines('a\nb\n', 'a\nb\n')).toEqual([]);
    });

    it('ignores line ending differences', () => {
        expect(changedLines('a\nb\n', 'a\r\nb\r\n')).toEqual([]);
    });

    it('marks an inserted line as added', () => {
        expect(changedLines('a\nc\n', 'a\nb\nc\n')).toEqual([{ line: 2, kind: 'added' }]);
    });

    it('marks a replaced line as modified', () => {
        expect(changedLines('a\nb\nc\n', 'a\nB\nc\n')).toEqual([{ line: 2, kind: 'modified' }]);
    });

    it('marks several changed regions', () => {
        expect(changedLines('a\nb\nc\nd\ne\n', 'a\nB\nc\nD\ne\n')).toEqual([
            { line: 2, kind: 'modified' },
            { line: 4, kind: 'modified' },
        ]);
    });

    it('reports no line for a deleted line', () => {
        expect(changedLines('a\nb\nc\n', 'a\nc\n')).toEqual([]);
    });

    it('treats every line as added when the baseline is empty', () => {
        expect(changedLines('', 'a\nb\n')).toEqual([
            { line: 1, kind: 'added' },
            { line: 2, kind: 'added' },
        ]);
    });

    it('marks a replaced block as modified and an inserted block as added', () => {
        expect(changedLines('a\nb\nc\n', 'a\nx\ny\nc\nd\n')).toEqual([
            { line: 2, kind: 'modified' },
            { line: 3, kind: 'modified' },
            { line: 5, kind: 'added' },
        ]);
    });
});

describe('diffLineKinds', () => {
    it('ignores the file headers before the first hunk', () => {
        const diff = [
            'diff --git a/file.txt b/file.txt',
            'index 111..222 100644',
            '--- a/file.txt',
            '+++ b/file.txt',
            '@@ -1,2 +1,2 @@',
            '-one',
            '+ONE',
            ' two',
        ].join('\n');
        expect(diffLineKinds(diff)).toEqual([
            { line: 5, kind: 'header' },
            { line: 6, kind: 'deleted' },
            { line: 7, kind: 'added' },
        ]);
    });

    it('treats a body line beginning with +++ as an addition', () => {
        expect(diffLineKinds('@@ -1 +1 @@\n+++ added marker')).toEqual([
            { line: 1, kind: 'header' },
            { line: 2, kind: 'added' },
        ]);
    });

    it('finds every hunk in a multi-hunk diff', () => {
        const diff = ['@@ -1 +1 @@', '-a', '+A', '@@ -9 +9 @@', '-b', '+B'].join('\r\n');
        expect(diffLineKinds(diff)).toEqual([
            { line: 1, kind: 'header' },
            { line: 2, kind: 'deleted' },
            { line: 3, kind: 'added' },
            { line: 4, kind: 'header' },
            { line: 5, kind: 'deleted' },
            { line: 6, kind: 'added' },
        ]);
    });

    it('tints nothing when there is no hunk', () => {
        expect(diffLineKinds('No changes against HEAD for file.txt.\n')).toEqual([]);
    });
});
