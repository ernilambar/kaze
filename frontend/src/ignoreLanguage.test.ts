import { describe, expect, it } from 'vitest';
import { IGNORE_NAME } from './ignoreLanguage';

// languageForPath lowercases the basename before testing, so the test does too.
describe('IGNORE_NAME', () => {
    it.each([
        '.gitignore',
        '.gitignore_global',
        '.gitignore-global',
        '.dockerignore',
        '.npmignore',
        '.eslintignore',
        '.prettierignore',
        '.stylelintignore',
        '.helmignore',
        '.vercelignore',
        '.vscodeignore',
        '.ignore',
        '.GITIGNORE',
    ])('treats %s as an ignore file', (name) => {
        expect(IGNORE_NAME.test(name.toLowerCase())).toBe(true);
    });

    it.each([
        '.env',
        '.gitignore.bak',
        '.ignore.bak',
        '.git-blame-ignore-revs',
        'gitignore',
        'ignore',
        'notes.txt',
    ])('does not treat %s as an ignore file', (name) => {
        expect(IGNORE_NAME.test(name.toLowerCase())).toBe(false);
    });
});
