import { describe, expect, it } from 'vitest';
import { formatCursorPosition, indentationLabel, lineEndingLabel, noticeFor } from './status';

describe('formatCursorPosition', () => {
    it.each([
        [1, 1, 0, 'Ln 1, Col 1'],
        [12, 34, 0, 'Ln 12, Col 34'],
        [1200, 80, 0, 'Ln 1200, Col 80'],
        [3, 7, 12, 'Ln 3, Col 7 (12 selected)'],
        [3, 7, 1, 'Ln 3, Col 7 (1 selected)'],
    ])('formats line %i column %i selection %i', (line, column, selected, expected) => {
        expect(formatCursorPosition(line, column, selected)).toBe(expected);
    });
});

describe('lineEndingLabel', () => {
    it('names both sequences the editor writes', () => {
        expect(lineEndingLabel('lf')).toBe('LF');
        expect(lineEndingLabel('crlf')).toBe('CRLF');
    });
});

describe('indentationLabel', () => {
    it('names the indent unit a buffer inserts', () => {
        expect(indentationLabel(true, 2, 4)).toBe('Spaces: 2');
        expect(indentationLabel(true, 8, 8)).toBe('Spaces: 8');
    });

    it('names the width a tab renders at', () => {
        expect(indentationLabel(false, 4, 4)).toBe('Tab Size: 4');
        expect(indentationLabel(false, 2, 8)).toBe('Tab Size: 8');
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
