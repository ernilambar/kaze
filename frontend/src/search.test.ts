import { describe, expect, it } from 'vitest';
import type { SearchMatch } from '../bindings/kova/models';
import { groupSearchMatches, searchSummary } from './search';

function match(path: string, line: number): SearchMatch {
    return { path, name: path.split('/').pop() ?? path, line, column: 1, text: 'hit' };
}

describe('groupSearchMatches', () => {
    it('collects matches under their file in reported order', () => {
        const groups = groupSearchMatches([
            match('/a.ts', 2),
            match('/b.ts', 5),
            match('/a.ts', 9),
        ]);

        expect(groups).toHaveLength(2);
        expect(groups[0].name).toBe('a.ts');
        expect(groups[0].matches.map((entry) => entry.line)).toEqual([2, 9]);
        expect(groups[1].name).toBe('b.ts');
    });

    it('is empty without matches', () => {
        expect(groupSearchMatches([])).toEqual([]);
    });
});

describe('searchSummary', () => {
    it('reports counts with singular forms', () => {
        expect(searchSummary(1, 1, false)).toBe('1 match in 1 file');
        expect(searchSummary(2, 5, false)).toBe('5 matches in 2 files');
    });

    it('marks a truncated result', () => {
        expect(searchSummary(1, 500, true)).toBe('500 matches in 1 file (limited)');
    });

    it('says so when nothing matched', () => {
        expect(searchSummary(0, 0, false)).toBe('No results.');
    });
});
