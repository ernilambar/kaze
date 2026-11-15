import { describe, expect, it } from 'vitest';
import { appendFinalNewline, trimTrailingWhitespace } from './text';

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
