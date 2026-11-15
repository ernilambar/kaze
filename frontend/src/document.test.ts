import { describe, expect, it } from 'vitest';
import {
    createUntitledDocument,
    emptyLaunchRequest,
    nextUntitledName,
    shownName,
    toBufferDocument,
    toOpenDocument,
    type OpenDocument,
} from './document';

function document(overrides: Partial<OpenDocument> = {}): OpenDocument {
    return {
        id: '/tmp/notes.txt',
        path: '/tmp/notes.txt',
        name: 'notes.txt',
        kind: 'text',
        content: '',
        src: '',
        modified: 0,
        dirty: false,
        ...overrides,
    };
}

describe('shownName', () => {
    it('falls back to an untitled label before a file is opened', () => {
        expect(shownName(undefined)).toBe('Untitled');
    });

    it('uses the document name once one is open', () => {
        expect(shownName(document())).toBe('notes.txt');
    });
});

describe('toOpenDocument', () => {
    it('maps the generated binding model and starts clean', () => {
        const model = {
            path: '/tmp/notes.txt',
            name: 'notes.txt',
            kind: 'text',
            content: 'hello',
            src: '',
            modified: 1712345678901,
        };

        expect(toOpenDocument(model)).toEqual({
            id: '/tmp/notes.txt',
            path: '/tmp/notes.txt',
            name: 'notes.txt',
            kind: 'text',
            content: 'hello',
            src: '',
            modified: 1712345678901,
            dirty: false,
        });
    });

    it('keeps an image data URL and marks the tab as an image', () => {
        const model = {
            path: '/tmp/logo.png',
            name: 'logo.png',
            kind: 'image',
            content: '',
            src: 'data:image/png;base64,AAAA',
            modified: 0,
        };

        expect(toOpenDocument(model)).toMatchObject({
            id: '/tmp/logo.png',
            kind: 'image',
            src: 'data:image/png;base64,AAAA',
        });
    });
});

function untitled(name: string): OpenDocument {
    return document({ id: `untitled:${name}`, path: '', name });
}

describe('nextUntitledName', () => {
    it('starts at one', () => {
        expect(nextUntitledName([])).toBe('Untitled-1');
    });

    it('continues after the highest number open', () => {
        expect(nextUntitledName([untitled('Untitled-1'), untitled('Untitled-2')])).toBe('Untitled-3');
    });

    it('does not reuse a number while a higher one is open', () => {
        expect(nextUntitledName([untitled('Untitled-2')])).toBe('Untitled-3');
    });

    it('ignores saved files with other names', () => {
        expect(nextUntitledName([document()])).toBe('Untitled-1');
    });
});

describe('createUntitledDocument', () => {
    it('is an empty buffer with no path and an id that is not a path', () => {
        expect(createUntitledDocument([document()])).toEqual({
            id: 'untitled:Untitled-1',
            path: '',
            name: 'Untitled-1',
            kind: 'text',
            content: '',
            src: '',
            modified: 0,
            dirty: false,
        });
    });
});

describe('emptyLaunchRequest', () => {
    it('starts with array fields so the UI never reads null', () => {
        expect(emptyLaunchRequest).toEqual({
            files: [],
            folders: [],
            missing: [],
            buffers: [],
            replace_folders: false,
            restore_session: false,
        });
    });
});

describe('toBufferDocument', () => {
    it('keeps a dirty buffer text and path when it moves windows', () => {
        expect(
            toBufferDocument({
                name: 'notes.txt',
                path: '/tmp/notes.txt',
                kind: 'text',
                content: 'unsaved',
                saved_content: 'on disk',
                modified: 42,
                dirty: true,
                source_path: '',
            }),
        ).toEqual({
            id: '/tmp/notes.txt',
            path: '/tmp/notes.txt',
            name: 'notes.txt',
            kind: 'text',
            content: 'unsaved',
            savedContent: 'on disk',
            src: '',
            modified: 42,
            dirty: true,
        });
    });

    it('gives an untitled buffer its name as an id', () => {
        expect(
            toBufferDocument({
                name: 'Untitled-2',
                path: '',
                kind: 'text',
                content: 'draft',
                saved_content: '',
                modified: 0,
                dirty: true,
                source_path: '',
            }).id,
        ).toBe('untitled:Untitled-2');
    });

    it('rebuilds a diff tab from the file it shows', () => {
        const moved = toBufferDocument({
            name: 'notes.txt (Diff)',
            path: '',
            kind: 'diff',
            content: '@@ -1 +1 @@',
            saved_content: '',
            modified: 0,
            dirty: false,
            source_path: '/tmp/notes.txt',
        });
        expect(moved.id).toBe('diff:/tmp/notes.txt');
        expect(moved.sourcePath).toBe('/tmp/notes.txt');
    });
});
