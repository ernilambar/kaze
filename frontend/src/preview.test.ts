import { describe, expect, it } from 'vitest';
import { previewKindFor, svgDataUrl } from './preview';

describe('previewKindFor', () => {
    it.each([
        ['notes.md', 'markdown'],
        ['README.MARKDOWN', 'markdown'],
        ['docs/guide.mkd', 'markdown'],
        ['icon.svg', 'svg'],
        ['/tmp/logo.SVG', 'svg'],
        ['notes.txt', ''],
        ['', ''],
    ])('previews %s as %s', (path, expected) => {
        expect(previewKindFor(path)).toBe(expected);
    });
});

describe('svgDataUrl', () => {
    it('embeds the document as an escaped data URL', () => {
        expect(svgDataUrl('<svg/>')).toBe('data:image/svg+xml;charset=utf-8,%3Csvg%2F%3E');
    });
});
