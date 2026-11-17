import { describe, expect, it } from 'vitest';
import {
    errorMessage,
    folderName,
    isWithin,
    relativePath,
    sidebarLabel,
    toFolderEntry,
} from './sidebar';

describe('folderName', () => {
    it.each([
        ['', ''],
        ['/', '/'],
        ['/Users/me/projects', 'projects'],
        ['/Users/me/projects/', 'projects'],
    ])('labels %j as %j', (folder, expected) => {
        expect(folderName(folder)).toBe(expected);
    });
});

describe('sidebarLabel', () => {
    it('is empty without roots', () => {
        expect(sidebarLabel([])).toBe('');
    });

    it('names a single root', () => {
        expect(sidebarLabel(['/Users/me/projects'])).toBe('projects');
    });

    it('counts several roots instead of naming only the first', () => {
        expect(sidebarLabel(['/one', '/two', '/three'])).toBe('3 folders');
    });
});

describe('toFolderEntry', () => {
    it('maps the generated binding model', () => {
        const model = { path: '/tmp/src', name: 'src', is_dir: true };

        expect(toFolderEntry(model)).toEqual({ path: '/tmp/src', name: 'src', isDir: true });
    });
});

describe('isWithin', () => {
    it.each([
        ['/projects', '/projects/a.txt', true],
        ['/projects', '/projects/src', true],
        ['/projects/', '/projects/a.txt', true],
        ['/projects', '/projects-old/a.txt', false],
        ['/projects', '/projects', false],
        ['', '/projects/a.txt', false],
        ['/projects', '', false],
    ])('reports whether %j contains %j', (folder, path, expected) => {
        expect(isWithin(folder, path)).toBe(expected);
    });
});

describe('errorMessage', () => {
    it('keeps a message the backend sent as a string', () => {
        expect(errorMessage('/tmp/notes.txt is not a folder')).toBe('/tmp/notes.txt is not a folder');
    });

    it('falls back to an Error message', () => {
        expect(errorMessage(new Error('dialog failed'))).toBe('dialog failed');
    });

    it('never renders an empty notice', () => {
        expect(errorMessage(undefined)).toBe('Something went wrong');
        expect(errorMessage('')).toBe('Something went wrong');
    });
});

describe('relativePath', () => {
    it('removes the innermost root', () => {
        expect(relativePath('/projects/app/src/app.ts', ['/projects', '/projects/app'])).toBe(
            'src/app.ts',
        );
    });

    it('returns a path outside every root unchanged', () => {
        expect(relativePath('/elsewhere/app.ts', ['/projects'])).toBe('/elsewhere/app.ts');
    });
});
