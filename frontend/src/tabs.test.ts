import { describe, expect, it } from 'vitest';
import {
    closeDocumentTab,
    openDocumentTab,
    setDocumentDirty,
    updateDocument,
} from './tabs';
import type { OpenDocument } from './document';

function document(path: string): OpenDocument {
    return {
        id: path,
        path,
        name: path.split('/').pop() ?? path,
        kind: 'text',
        content: '',
        src: '',
        modified: 0,
        dirty: false,
    };
}

function untitled(name: string): OpenDocument {
    return {
        id: `untitled:${name}`,
        path: '',
        name,
        kind: 'text',
        content: '',
        src: '',
        modified: 0,
        dirty: false,
    };
}

const notes = document('/tmp/notes.txt');
const readme = document('/tmp/README.md');
const todo = document('/tmp/todo.md');

describe('openDocumentTab', () => {
    it('appends a document that is not open yet', () => {
        const result = openDocumentTab([notes], readme);

        expect(result.tabs).toEqual([notes, readme]);
        expect(result.activeId).toBe(readme.id);
    });

    it('keeps one tab per file and activates the one that is already open', () => {
        const reopened = { ...notes, content: 'changed on disk' };
        const result = openDocumentTab([notes, readme], reopened);

        expect(result.tabs).toEqual([notes, readme]);
        expect(result.activeId).toBe(notes.id);
    });

    it('keeps untitled buffers apart even though none has a path', () => {
        const first = untitled('Untitled-1');
        const second = untitled('Untitled-2');

        expect(openDocumentTab([first], second).tabs).toEqual([first, second]);
    });
});

describe('closeDocumentTab', () => {
    it('keeps the active tab when another tab is closed', () => {
        const result = closeDocumentTab([notes, readme], notes.id, readme.id);

        expect(result.tabs).toEqual([readme]);
        expect(result.activeId).toBe(readme.id);
    });

    it('activates the tab to the right when the active tab closes', () => {
        const result = closeDocumentTab([notes, readme, todo], readme.id, readme.id);

        expect(result.tabs).toEqual([notes, todo]);
        expect(result.activeId).toBe(todo.id);
    });

    it('falls back to the tab on the left when there is no tab to the right', () => {
        const result = closeDocumentTab([notes, readme], readme.id, readme.id);

        expect(result.tabs).toEqual([notes]);
        expect(result.activeId).toBe(notes.id);
    });

    it('clears the active id when the last tab closes', () => {
        const result = closeDocumentTab([notes], notes.id, notes.id);

        expect(result.tabs).toEqual([]);
        expect(result.activeId).toBe('');
    });

    it('closes an untitled buffer by its id', () => {
        const draft = untitled('Untitled-1');
        const result = closeDocumentTab([draft, notes], draft.id, draft.id);

        expect(result.tabs).toEqual([notes]);
        expect(result.activeId).toBe(notes.id);
    });

    it('ignores an id that is not open', () => {
        const result = closeDocumentTab([notes], todo.id, notes.id);

        expect(result.tabs).toEqual([notes]);
        expect(result.activeId).toBe(notes.id);
    });
});

describe('updateDocument', () => {
    it('patches only the named tab', () => {
        const result = updateDocument([notes, readme], notes.id, { path: '/tmp/moved.txt' });

        expect(result[0]?.path).toBe('/tmp/moved.txt');
        expect(result[1]).toBe(readme);
    });

    it('leaves the list alone when the tab is gone', () => {
        const tabs = [notes];

        expect(updateDocument(tabs, todo.id, { dirty: true })).toBe(tabs);
    });
});

describe('setDocumentDirty', () => {
    it('marks one tab modified', () => {
        const result = setDocumentDirty([notes, readme], notes.id, true);

        expect(result[0]?.dirty).toBe(true);
        expect(result[1]?.dirty).toBe(false);
    });

    it('returns the same list when the flag already matches', () => {
        const tabs = [notes, readme];

        expect(setDocumentDirty(tabs, notes.id, false)).toBe(tabs);
        expect(setDocumentDirty(tabs, todo.id, true)).toBe(tabs);
    });
});
