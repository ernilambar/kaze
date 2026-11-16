import { describe, expect, it } from 'vitest';
import {
    DRAG_TYPE,
    draggingPath,
    dropSource,
    entryName,
    fileDragPaths,
    parentPath,
    setDraggingPath,
    toFileTreeEntry,
} from './filetree';

describe('toFileTreeEntry', () => {
    it('maps the binding model', () => {
        expect(
            toFileTreeEntry({
                path: '/tmp/notes.txt',
                real_path: '/private/tmp/notes.txt',
                name: 'notes.txt',
                is_dir: false,
            }),
        ).toEqual({
            path: '/tmp/notes.txt',
            realPath: '/private/tmp/notes.txt',
            name: 'notes.txt',
            isDir: false,
        });
    });

    it('falls back to the written path when nothing was resolved', () => {
        const entry = toFileTreeEntry({
            path: '/tmp/new.txt',
            real_path: '',
            name: 'new.txt',
            is_dir: false,
        });

        expect(entry.realPath).toBe('/tmp/new.txt');
    });
});

describe('entryName', () => {
    it.each([
        ['/tmp/notes.txt', 'notes.txt'],
        ['/tmp/src', 'src'],
        ['/tmp/src/', 'src'],
        ['/tmp', 'tmp'],
        ['relative.txt', 'relative.txt'],
    ])('names %j as %j', (path, expected) => {
        expect(entryName(path)).toBe(expected);
    });
});

describe('parentPath', () => {
    it.each([
        ['/tmp/notes.txt', '/tmp'],
        ['/tmp/src/main.ts', '/tmp/src'],
        ['/notes.txt', '/'],
        ['/tmp/src/', '/tmp'],
    ])('puts %j in %j', (path, expected) => {
        expect(parentPath(path)).toBe(expected);
    });

    it('has no parent for a bare name', () => {
        expect(parentPath('notes.txt')).toBe('');
    });
});

describe('drag state', () => {
    it('remembers what a drag is moving', () => {
        setDraggingPath('/tmp/notes.txt');

        expect(draggingPath()).toBe('/tmp/notes.txt');

        setDraggingPath('');

        expect(draggingPath()).toBe('');
    });

    it('reads a path the tree itself put on the transfer', () => {
        const transfer = {
            getData: (type: string) => (type === DRAG_TYPE ? '/tmp/notes.txt' : ''),
            files: [],
        } as unknown as DataTransfer;

        expect(dropSource(transfer)).toBe('/tmp/notes.txt');
    });

    it('prefers the tree payload over a dragged file list', () => {
        const transfer = {
            getData: (type: string) => (type === DRAG_TYPE ? '/tmp/notes.txt' : ''),
            files: [{ path: '/tmp/other.txt' }],
        } as unknown as DataTransfer;

        expect(dropSource(transfer)).toBe('/tmp/notes.txt');
    });

    it('has nothing to move without a transfer', () => {
        expect(dropSource(null)).toBe('');
    });
});

describe('fileDragPaths', () => {
    it('reads the local path a webview puts on a dragged file', () => {
        const files = [{ path: '/tmp/a.txt' }, { path: '/tmp/b.txt' }] as unknown as FileList;

        expect(fileDragPaths(files)).toEqual(['/tmp/a.txt', '/tmp/b.txt']);
    });

    it('drops files whose path the webview does not expose', () => {
        const files = [{}, { path: '' }, { path: '/tmp/b.txt' }] as unknown as FileList;

        expect(fileDragPaths(files)).toEqual(['/tmp/b.txt']);
    });

    it('has no paths without a file list', () => {
        expect(fileDragPaths(null)).toEqual([]);
    });
});
