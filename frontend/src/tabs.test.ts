import { describe, expect, it } from 'vitest';
import { closeDocumentTab, openDocumentTab } from './tabs';
import type { OpenDocument } from './document';

function document(path: string): OpenDocument {
    return { path, name: path.split('/').pop() ?? path, kind: 'text', content: '', src: '' };
}

const notes = document('/tmp/notes.txt');
const readme = document('/tmp/README.md');
const todo = document('/tmp/todo.md');

describe('openDocumentTab', () => {
    it('appends a document that is not open yet', () => {
        expect(openDocumentTab([notes], readme)).toEqual([notes, readme]);
    });

    it('keeps one tab per path', () => {
        const reopened = { ...notes, content: 'changed on disk' };

        expect(openDocumentTab([notes, readme], reopened)).toEqual([notes, readme]);
    });
});

describe('closeDocumentTab', () => {
    it('keeps the active tab when another tab is closed', () => {
        const result = closeDocumentTab([notes, readme], notes.path, readme.path);

        expect(result.tabs).toEqual([readme]);
        expect(result.activePath).toBe(readme.path);
    });

    it('activates the tab to the right when the active tab closes', () => {
        const result = closeDocumentTab([notes, readme, todo], readme.path, readme.path);

        expect(result.tabs).toEqual([notes, todo]);
        expect(result.activePath).toBe(todo.path);
    });

    it('falls back to the tab on the left when there is no tab to the right', () => {
        const result = closeDocumentTab([notes, readme], readme.path, readme.path);

        expect(result.tabs).toEqual([notes]);
        expect(result.activePath).toBe(notes.path);
    });

    it('clears the active path when the last tab closes', () => {
        const result = closeDocumentTab([notes], notes.path, notes.path);

        expect(result.tabs).toEqual([]);
        expect(result.activePath).toBe('');
    });

    it('ignores a path that is not open', () => {
        const result = closeDocumentTab([notes], todo.path, notes.path);

        expect(result.tabs).toEqual([notes]);
        expect(result.activePath).toBe(notes.path);
    });
});
