import { describe, expect, it } from 'vitest';
import { shownName, toOpenDocument, emptyLaunchRequest } from './document';

describe('shownName', () => {
    it('falls back to the untitled buffer before a file is opened', () => {
        expect(shownName(undefined)).toBe('untitled.txt');
    });

    it('uses the document name once one is open', () => {
        expect(shownName({ path: '/tmp/notes.txt', name: 'notes.txt', content: '' })).toBe('notes.txt');
    });
});

describe('toOpenDocument', () => {
    it('maps the generated binding model', () => {
        const model = { path: '/tmp/notes.txt', name: 'notes.txt', content: 'hello' };

        expect(toOpenDocument(model)).toEqual({
            path: '/tmp/notes.txt',
            name: 'notes.txt',
            content: 'hello',
        });
    });
});

describe('emptyLaunchRequest', () => {
    it('starts with array fields so the UI never reads null', () => {
        expect(emptyLaunchRequest).toEqual({ files: [], folders: [], missing: [] });
    });
});
