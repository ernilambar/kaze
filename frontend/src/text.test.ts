import { describe, expect, it } from 'vitest';
import { appendFinalNewline, transformLines, trimTrailingWhitespace } from './text';

describe('trimTrailingWhitespace', () => {
    it('removes spaces and tabs at line ends only', () => {
        expect(trimTrailingWhitespace('one  \ntwo\t\n  three')).toBe('one\ntwo\n  three');
    });

    it('leaves interior whitespace alone', () => {
        expect(trimTrailingWhitespace('a  b\nc\n')).toBe('a  b\nc\n');
    });
});

describe('appendFinalNewline', () => {
    it('adds the newline that is missing', () => {
        expect(appendFinalNewline('one\ntwo')).toBe('one\ntwo\n');
    });

    it('does not add a second newline', () => {
        expect(appendFinalNewline('one\n')).toBe('one\n');
    });

    it('keeps an empty buffer empty', () => {
        expect(appendFinalNewline('')).toBe('');
    });
});

describe('transformLines', () => {
    it('sorts lines in both directions and keeps the final newline', () => {
        expect(transformLines('b\na\nc\n', 'sort-lines-ascending')).toBe('a\nb\nc\n');
        expect(transformLines('b\na\nc\n', 'sort-lines-descending')).toBe('c\nb\na\n');
    });

    it('keeps the first occurrence of every line', () => {
        expect(transformLines('a\nb\na\n', 'delete-duplicate-lines')).toBe('a\nb\n');
    });

    it('reverses lines without moving the final newline', () => {
        expect(transformLines('a\nb\nc', 'reverse-lines')).toBe('c\nb\na');
        expect(transformLines('a\nb\n', 'reverse-lines')).toBe('b\na\n');
    });

    it('leaves an empty buffer empty', () => {
        expect(transformLines('', 'sort-lines-ascending')).toBe('');
    });
});
