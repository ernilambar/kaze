import { describe, expect, it } from 'vitest';
import { formatCursorPosition, noticeFor } from './status';

describe('formatCursorPosition', () => {
    it.each([
        [1, 1, 'Ln 1, Col 1'],
        [12, 34, 'Ln 12, Col 34'],
        [1200, 80, 'Ln 1200, Col 80'],
    ])('formats line %i column %i', (line, column, expected) => {
        expect(formatCursorPosition(line, column)).toBe(expected);
    });
});

describe('noticeFor', () => {
    it('says nothing when everything opened', () => {
        expect(noticeFor({ missing: [] })).toBe('');
    });

    it('names a single missing path', () => {
        expect(noticeFor({ missing: ['notes.txt'] })).toBe('Cannot open notes.txt');
    });

    it('counts several missing paths', () => {
        expect(noticeFor({ missing: ['a.txt', 'b.txt'] })).toBe('Cannot open 2 items');
    });
});
