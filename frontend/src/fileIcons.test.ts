import { describe, expect, it } from 'vitest';
import raw from './fileIcons.json';
import { fileIcon } from './fileIcons';

interface IconData {
    glyphs: Record<string, [string, string, number, number, number]>;
    rules: [string, string, string, string | null][];
}

const { glyphs, rules } = raw as unknown as IconData;

function glyphOf(icon: string): [string, string] {
    const glyph = glyphs[icon];
    return [glyph[0], glyph[1]];
}

describe('fileIcon', () => {
    it.each([
        ['app.js', 'js-icon'],
        ['main.ts', 'ts-icon'],
        ['script.py', 'python-icon'],
        ['lib.rs', 'rust-icon'],
        ['index.html', 'html5-icon'],
        ['style.css', 'css3-icon'],
        ['README.md', 'book-icon'],
        ['package.json', 'npm-icon'],
        ['App.svelte', 'svelte-icon'],
    ])('maps %s to the %s glyph', (name, icon) => {
        const actual = fileIcon(name);

        expect([actual.glyph, actual.family]).toEqual(glyphOf(icon));
    });

    it('matches exact filenames before their extension', () => {
        expect([fileIcon('Dockerfile').glyph, fileIcon('Dockerfile').family]).toEqual(
            glyphOf('docker-icon'),
        );
        expect([fileIcon('Makefile').glyph, fileIcon('Makefile').family]).toEqual(
            glyphOf('checklist-icon'),
        );
    });

    it('prefers the test-file icon over the language icon', () => {
        expect([fileIcon('foo.test.ts').glyph, fileIcon('foo.test.ts').family]).toEqual(
            glyphOf('test-ts-icon'),
        );
    });

    it('ignores case', () => {
        expect(fileIcon('MAIN.TS')).toEqual(fileIcon('main.ts'));
    });

    it('falls back to a muted file outline for unknown types', () => {
        expect(fileIcon('archive.unknown')).toEqual({
            glyph: '\uf016',
            family: 'FontAwesome',
            size: 15,
            top: 0,
            left: 0,
            color: 'var(--fg-muted)',
        });
    });

    it('keeps the mapping data self-consistent', () => {
        const missing = rules.filter(([, , icon]) => glyphs[icon] === undefined);

        expect(missing).toEqual([]);
    });
});
