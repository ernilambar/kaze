import { describe, expect, it } from 'vitest';
import type { SearchMatch } from '../bindings/kaze/models';
import {
    expectedModified,
    groupSearchMatches,
    replaceConfirmMessage,
    replaceNotice,
    replaceText,
    searchExpression,
    searchSummary,
    stepSearchResult,
} from './search';

function match(path: string, line: number, modified = 1000): SearchMatch {
    return {
        path,
        name: path.split('/').pop() ?? path,
        line,
        column: 1,
        text: 'hit',
        modified,
    };
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

describe('stepSearchResult', () => {
    it('starts at the first result going forward and the last going back', () => {
        expect(stepSearchResult(-1, 3, 1)).toBe(0);
        expect(stepSearchResult(-1, 3, -1)).toBe(2);
    });

    it('advances and rewinds, wrapping at both ends', () => {
        expect(stepSearchResult(0, 3, 1)).toBe(1);
        expect(stepSearchResult(2, 3, 1)).toBe(0);
        expect(stepSearchResult(0, 3, -1)).toBe(2);
    });

    it('reports no step without results', () => {
        expect(stepSearchResult(-1, 0, 1)).toBe(-1);
    });
});

describe('expectedModified', () => {
    it('maps each matching file to the mtime the search read', () => {
        expect(
            expectedModified([match('/a.ts', 1, 111), match('/a.ts', 4, 111), match('/b.ts', 2, 222)]),
        ).toEqual({ '/a.ts': 111, '/b.ts': 222 });
    });
});

describe('searchExpression and replaceText', () => {
    it('treats a literal query as text, including $ in the replacement', () => {
        const expression = searchExpression('a.b', false, false);
        expect(replaceText('a.b aXb A.B', expression, '$&', false)).toBe('$& aXb $&');
    });

    it('honours case sensitivity and regex capture groups', () => {
        const expression = searchExpression('(a)(b)', true, true);
        expect(replaceText('ab AB', expression, '$2$1', true)).toBe('ba AB');
    });
});

describe('replaceConfirmMessage', () => {
    it('states the match and file counts', () => {
        expect(replaceConfirmMessage(5, 2, false)).toContain('5 matches in 2 files');
        expect(replaceConfirmMessage(1, 1, false)).toContain('1 match in 1 file');
    });

    it('warns when the result list was limited', () => {
        expect(replaceConfirmMessage(500, 3, true)).toContain('counts shown are a minimum');
    });
});

describe('replaceNotice', () => {
    it('reports the replacement count', () => {
        expect(replaceNotice(3, 2, [])).toBe('Replaced 3 in 2 files.');
    });

    it('names files skipped because they changed on disk', () => {
        const notice = replaceNotice(1, 1, ['/a.ts', '/b.ts']);
        expect(notice).toContain('Skipped 2 files');
        expect(notice).toContain('/a.ts, /b.ts');
    });
});
