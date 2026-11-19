import { describe, expect, it } from 'vitest';
import { languageForPath } from './editor';

describe('languageForPath', () => {
    it('reads a .dist template by the name it wraps', () => {
        expect(languageForPath('/w/phpunit.xml.dist')).toBe('xml');
        expect(languageForPath('/w/package.json.dist')).toBe('json');
        expect(languageForPath('/w/config.yaml.dist')).toBe('yaml');
        expect(languageForPath('/w/app.ts.dist')).toBe('typescript');
    });

    it('ignores case', () => {
        expect(languageForPath('C:\\w\\config.XML.DIST')).toBe('xml');
    });

    it('resolves the wrapped name when it has no extension', () => {
        // Monaco registers `config` as an ini filename.
        expect(languageForPath('/w/config.dist')).toBe(languageForPath('/w/config'));
    });

    it('keeps names with no wrapped language plain text', () => {
        expect(languageForPath('/w/.dist')).toBe('plaintext');
        expect(languageForPath('/w/notes.txt.dist')).toBe('plaintext');
    });
});
