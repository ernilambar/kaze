import { describe, expect, it } from 'vitest';
import raw from './fileIcons.json';
import { fileIcon, folderIcon } from './fileIcons';

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

    it('gives .dist templates the icon of the name they wrap', () => {
        expect(fileIcon('phpunit.xml.dist')).toEqual(fileIcon('phpunit.xml'));
        expect(fileIcon('config.xml.dist')).toEqual(fileIcon('config.xml'));
        expect(fileIcon('package.json.dist')).toEqual(fileIcon('package.json'));
        expect(fileIcon('app.ts.dist')).toEqual(fileIcon('app.ts'));
        expect(fileIcon('config.XML.DIST')).toEqual(fileIcon('config.xml'));
    });

    it('falls back when nothing is left to wrap', () => {
        expect(fileIcon('config.dist')).toEqual(fileIcon('config'));
        expect(fileIcon('.dist')).toEqual(fileIcon('unknown-file'));
    });

    it('ignores case', () => {
        expect(fileIcon('MAIN.TS')).toEqual(fileIcon('main.ts'));
    });

    it('draws markdown with the Devicons markdown glyph', () => {
        const notes = fileIcon('notes.md');

        expect([notes.glyph, notes.family]).toEqual(['\ue63e', 'Devicons']);
        expect(fileIcon('README.md')).toEqual(notes);
        expect(fileIcon('readme.markdown')).toEqual(notes);
        expect([fileIcon('README').glyph, fileIcon('README').family]).toEqual(
            glyphOf('book-icon'),
        );
    });

    it('uses a substituted glyph for icons the bundled font lacks', () => {
        const env = fileIcon('.env');

        expect([env.glyph, env.family]).toEqual(glyphOf('config-icon'));
        expect(env.color).toBe('#ee9e2e');
        expect([fileIcon('windi.config.ts').glyph, fileIcon('windi.config.ts').family]).toEqual(
            glyphOf('tailwind-icon'),
        );
    });

    it('gives every dotenv name the dotenv icon', () => {
        const env = fileIcon('.env');

        for (const name of [
            '.env.local',
            '.env.production.example',
            '.ENV.LOCAL',
            'prod.env',
            'app.prod.env',
        ]) {
            expect(fileIcon(name)).toEqual(env);
        }
        expect(fileIcon('.envrc')).not.toEqual(env);
        expect(fileIcon('config.env.example')).not.toEqual(env);
    });

    it('keeps .bak files on the backup icon', () => {
        expect(fileIcon('.env.bak')).toEqual(fileIcon('archive.bak'));
        expect(fileIcon('.env.local.bak')).toEqual(fileIcon('archive.bak'));
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

describe('folderIcon', () => {
    it('gives .github the GitHub mark', () => {
        expect(folderIcon('.github')).toEqual({
            glyph: '\uf09b',
            family: 'FontAwesome',
            size: 15,
            top: 0,
            left: 0,
            color: 'var(--fg)',
        });
        expect(folderIcon('.GitHub')).toEqual(folderIcon('.github'));
    });

    it('gives node_modules the Node.js mark', () => {
        expect(folderIcon('node_modules')).toEqual({
            glyph: '\ue618',
            family: 'Devicons',
            size: 16,
            top: 3,
            left: 0,
            color: '#90a959',
        });
    });

    it('leaves other folders to the plain folder shape', () => {
        expect(folderIcon('src')).toBeUndefined();
        expect(folderIcon('.git')).toBeUndefined();
    });
});
