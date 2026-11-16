import { describe, expect, it } from 'vitest';
import {
    closeDocumentTab,
    findDocumentByPath,
    moveDocumentTab,
    openDocumentTab,
    pathWithin,
    retargetDocumentPath,
    retargetDocumentPaths,
    setDocumentDirty,
    stepTabId,
    tabIdAt,
    updateDocument,
} from './tabs';
import type { OpenDocument } from './document';

function document(path: string): OpenDocument {
    return {
        id: path,
        path,
        realPath: path,
        name: path.split('/').pop() ?? path,
        kind: 'text',
        content: '',
        src: '',
        modified: 0,
        dirty: false,
        bom: false,
        eol: 'lf',
        invalidUTF8: false,
        deleted: false,
    };
}

// linked is a tab opened through a symlink: the spelling is the link, the
// identity is the target.
function linked(alias: string, target: string): OpenDocument {
    const tab = document(target);
    return { ...tab, id: alias, path: alias, realPath: target, name: alias.split('/').pop() ?? alias };
}

function untitled(name: string): OpenDocument {
    return {
        id: `untitled:${name}`,
        path: '',
        realPath: '',
        name,
        kind: 'text',
        content: '',
        src: '',
        modified: 0,
        dirty: false,
        bom: false,
        eol: 'lf',
        invalidUTF8: false,
        deleted: false,
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

    it('opens one tab for a file reached through a symlink and directly', () => {
        const viaLink = linked('/tmp/project-link/notes.txt', '/tmp/project/notes.txt');
        const direct = document('/tmp/project/notes.txt');

        expect(openDocumentTab([viaLink], direct)).toEqual({
            tabs: [viaLink],
            activeId: viaLink.id,
        });
        expect(openDocumentTab([direct], viaLink)).toEqual({
            tabs: [direct],
            activeId: direct.id,
        });
    });
});

describe('findDocumentByPath', () => {
    it('matches the spelling a tab was opened with and its resolved path', () => {
        const viaLink = linked('/tmp/project-link/notes.txt', '/tmp/project/notes.txt');

        expect(findDocumentByPath([viaLink], '/tmp/project-link/notes.txt')).toBe(viaLink);
        expect(findDocumentByPath([viaLink], '/tmp/project/notes.txt')).toBe(viaLink);
    });

    it('never matches an empty path', () => {
        expect(findDocumentByPath([untitled('Untitled-1')], '')).toBeUndefined();
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

describe('moveDocumentTab', () => {
    const tabs = [notes, readme, todo];

    it('drops a tab after the target tab', () => {
        const result = moveDocumentTab(tabs, notes.id, readme.id, true);

        expect(result).toEqual([readme, notes, todo]);
    });

    it('drops a tab before the target tab', () => {
        const result = moveDocumentTab(tabs, todo.id, notes.id, false);

        expect(result).toEqual([todo, notes, readme]);
    });

    it('moves a tab to the far end', () => {
        const result = moveDocumentTab(tabs, notes.id, todo.id, true);

        expect(result).toEqual([readme, todo, notes]);
    });

    it('keeps the list when the drop would not move the tab', () => {
        expect(moveDocumentTab(tabs, notes.id, readme.id, false)).toBe(tabs);
        expect(moveDocumentTab(tabs, readme.id, notes.id, true)).toBe(tabs);
    });

    it('ignores a tab or target that is not open', () => {
        expect(moveDocumentTab(tabs, todo.id, todo.id, true)).toBe(tabs);
        expect(moveDocumentTab(tabs, document('/tmp/gone.txt').id, notes.id, true)).toBe(tabs);
        expect(moveDocumentTab(tabs, notes.id, document('/tmp/gone.txt').id, true)).toBe(tabs);
    });

    it('leaves the source order alone', () => {
        moveDocumentTab(tabs, notes.id, todo.id, true);

        expect(tabs).toEqual([notes, readme, todo]);
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

describe('stepTabId', () => {
    const tabs = [notes, readme, todo];

    it('moves forward and wraps at the end', () => {
        expect(stepTabId(tabs, notes.id, 1)).toBe(readme.id);
        expect(stepTabId(tabs, todo.id, 1)).toBe(notes.id);
    });

    it('moves backward and wraps at the start', () => {
        expect(stepTabId(tabs, readme.id, -1)).toBe(notes.id);
        expect(stepTabId(tabs, notes.id, -1)).toBe(todo.id);
    });

    it('starts from the first tab when none is active', () => {
        expect(stepTabId(tabs, '', 1)).toBe(notes.id);
    });

    it('keeps the current id when there are no tabs', () => {
        expect(stepTabId([], 'active', 1)).toBe('active');
    });
});

describe('tabIdAt', () => {
    const tabs = [notes, readme, todo];

    it('maps a 1-based position', () => {
        expect(tabIdAt(tabs, 1)).toBe(notes.id);
        expect(tabIdAt(tabs, 3)).toBe(todo.id);
    });

    it('is empty outside the list', () => {
        expect(tabIdAt(tabs, 0)).toBe('');
        expect(tabIdAt(tabs, 4)).toBe('');
    });
});

describe('pathWithin', () => {
    it.each([
        ['/a/b', '/a/b', true],
        ['/a/b/c', '/a/b', true],
        ['/a/bc', '/a/b', false],
        ['/a', '/a/b', false],
        ['', '/a', false],
    ])('matches %j under %j as %j', (path, base, expected) => {
        expect(pathWithin(path, base)).toBe(expected);
    });

    it('accepts either separator', () => {
        expect(pathWithin('C:\\work\\notes.txt', 'C:/work')).toBe(true);
    });
});

describe('retargetDocumentPath', () => {
    it('renames the tab that holds the file', () => {
        const moved = retargetDocumentPath(notes, '/tmp/notes.txt', '/tmp/journal.txt');

        expect(moved?.path).toBe('/tmp/journal.txt');
        expect(moved?.realPath).toBe('/tmp/journal.txt');
        expect(moved?.name).toBe('journal.txt');
        // The id travels with the buffer, so the editor keeps its model.
        expect(moved?.id).toBe(notes.id);
    });

    it('keeps a dirty buffer dirty', () => {
        const dirty: OpenDocument = { ...notes, dirty: true };

        expect(retargetDocumentPath(dirty, '/tmp/notes.txt', '/tmp/journal.txt')?.dirty).toBe(true);
    });

    it('moves the tabs of files inside a renamed folder', () => {
        const nested: OpenDocument = {
            ...notes,
            id: '/src/main.ts',
            path: '/src/main.ts',
            realPath: '/src/main.ts',
            name: 'main.ts',
        };

        expect(retargetDocumentPath(nested, '/src', '/lib')?.path).toBe('/lib/main.ts');
    });

    it('matches the resolved path too', () => {
        const linked: OpenDocument = {
            ...notes,
            realPath: '/private/notes.txt',
        };

        const moved = retargetDocumentPath(linked, '/private/notes.txt', '/private/journal.txt');

        expect(moved?.path).toBe('/private/journal.txt');
        expect(moved?.id).toBe(linked.id);
    });

    it('leaves unrelated tabs and untitled buffers alone', () => {
        expect(retargetDocumentPath(notes, '/tmp/other.txt', '/tmp/moved.txt')).toBeUndefined();
        const buffer = untitled('Untitled-1');
        expect(retargetDocumentPath(buffer, '/tmp/notes.txt', '/tmp/journal.txt')).toBeUndefined();
    });

    it('does nothing when the path did not change', () => {
        expect(retargetDocumentPath(notes, '/tmp/notes.txt', '/tmp/notes.txt')).toBeUndefined();
    });

    it('follows the file a diff tab shows', () => {
        const diff: OpenDocument = {
            ...notes,
            id: 'diff:/tmp/notes.txt',
            path: '',
            realPath: '',
            name: 'notes.txt (Diff)',
            kind: 'diff',
            sourcePath: '/tmp/notes.txt',
        };

        const moved = retargetDocumentPath(diff, '/tmp/notes.txt', '/tmp/journal.txt');

        expect(moved?.sourcePath).toBe('/tmp/journal.txt');
        expect(moved?.id).toBe('diff:/tmp/journal.txt');
        expect(moved?.name).toBe('journal.txt (Diff)');
    });
});

describe('retargetDocumentPaths', () => {
    it('returns the same list when nothing matched', () => {
        const tabs = [notes, readme];

        expect(retargetDocumentPaths(tabs, [{ from: '/other', to: '/moved' }])).toBe(tabs);
        expect(retargetDocumentPaths(tabs, [])).toBe(tabs);
    });

    it('applies a move to every tab it touches', () => {
        const moved = retargetDocumentPaths(
            [notes, readme],
            [{ from: '/tmp/notes.txt', to: '/tmp/journal.txt' }],
        );

        expect(moved[0]?.path).toBe('/tmp/journal.txt');
        expect(moved[1]).toBe(readme);
    });
});
