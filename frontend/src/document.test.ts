import { describe, expect, it } from 'vitest';
import {
    createUntitledDocument,
    emptyLaunchRequest,
    nextUntitledName,
    normalizeEol,
    shownName,
    toBufferDocument,
    toOpenDocument,
    type OpenDocument,
} from './document';

function document(overrides: Partial<OpenDocument> = {}): OpenDocument {
    return {
        id: '/tmp/notes.txt',
        path: '/tmp/notes.txt',
        realPath: '/tmp/notes.txt',
        name: 'notes.txt',
        kind: 'text',
        content: '',
        src: '',
        modified: 0,
        dirty: false,
        bom: false,
        eol: 'lf',
        invalidUTF8: false,
        deleted: false,
        ...overrides,
    };
}

describe('normalizeEol', () => {
    it('converts every line ending to the document sequence', () => {
        expect(normalizeEol('a\nb\r\nc\r', 'crlf')).toBe('a\r\nb\r\nc\r\n');
        expect(normalizeEol('a\nb\r\nc\r', 'lf')).toBe('a\nb\nc\n');
    });
});

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
            real_path: '/tmp/notes.txt',
            name: 'notes.txt',
            kind: 'text',
            content: 'hello',
            src: '',
            modified: 1712345678901,
            bom: true,
            eol: 'crlf',
            invalid_utf8: false,
        };

        expect(toOpenDocument(model)).toEqual({
            id: '/tmp/notes.txt',
            path: '/tmp/notes.txt',
            realPath: '/tmp/notes.txt',
            name: 'notes.txt',
            kind: 'text',
            content: 'hello',
            src: '',
            modified: 1712345678901,
            dirty: false,
            bom: true,
            eol: 'crlf',
            invalidUTF8: false,
            deleted: false,
        });
    });

    it('keeps the opened spelling but records the resolved path', () => {
        const model = {
            path: '/tmp/alias.txt',
            real_path: '/tmp/notes.txt',
            name: 'alias.txt',
            kind: 'text',
            content: 'hello',
            src: '',
            modified: 0,
            bom: false,
            eol: 'lf',
            invalid_utf8: false,
        };

        const opened = toOpenDocument(model);
        expect(opened.id).toBe('/tmp/alias.txt');
        expect(opened.path).toBe('/tmp/alias.txt');
        expect(opened.realPath).toBe('/tmp/notes.txt');
    });

    it('keeps an image data URL and marks the tab as an image', () => {
        const model = {
            path: '/tmp/logo.png',
            real_path: '/tmp/logo.png',
            name: 'logo.png',
            kind: 'image',
            content: '',
            src: 'data:image/png;base64,AAAA',
            modified: 0,
            bom: false,
            eol: '',
            invalid_utf8: false,
        };

        expect(toOpenDocument(model)).toMatchObject({
            id: '/tmp/logo.png',
            kind: 'image',
            src: 'data:image/png;base64,AAAA',
        });
    });
});

function untitled(name: string): OpenDocument {
    return document({ id: `untitled:${name}`, path: '', realPath: '', name });
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
            realPath: '',
            name: 'Untitled-1',
            kind: 'text',
            content: '',
            src: '',
            modified: 0,
            dirty: false,
            bom: false,
            eol: 'lf',
            invalidUTF8: false,
            deleted: false,
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
                real_path: '/private/tmp/notes.txt',
                kind: 'text',
                content: 'unsaved',
                saved_content: 'on disk',
                modified: 42,
                dirty: true,
                source_path: '',
                bom: true,
                eol: 'crlf',
                invalid_utf8: false,
            }),
        ).toEqual({
            id: '/tmp/notes.txt',
            path: '/tmp/notes.txt',
            realPath: '/private/tmp/notes.txt',
            name: 'notes.txt',
            kind: 'text',
            content: 'unsaved',
            savedContent: 'on disk',
            src: '',
            modified: 42,
            dirty: true,
            bom: true,
            eol: 'crlf',
            invalidUTF8: false,
            deleted: false,
        });
    });

    it('gives an untitled buffer its name as an id', () => {
        expect(
            toBufferDocument({
                name: 'Untitled-2',
                path: '',
                real_path: '',
                kind: 'text',
                content: 'draft',
                saved_content: '',
                modified: 0,
                dirty: true,
                source_path: '',
                bom: false,
                eol: 'lf',
                invalid_utf8: false,
            }).id,
        ).toBe('untitled:Untitled-2');
    });

    it('rebuilds a diff tab from the file it shows', () => {
        const moved = toBufferDocument({
            name: 'notes.txt (Diff)',
            path: '',
            real_path: '',
            kind: 'diff',
            content: '@@ -1 +1 @@',
            saved_content: '',
            modified: 0,
            dirty: false,
            source_path: '/tmp/notes.txt',
            bom: false,
            eol: 'lf',
            invalid_utf8: false,
        });
        expect(moved.id).toBe('diff:/tmp/notes.txt');
        expect(moved.sourcePath).toBe('/tmp/notes.txt');
    });
});
