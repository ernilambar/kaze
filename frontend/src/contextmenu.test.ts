import { describe, expect, it } from 'vitest';
import { ROOT_LOCKED_COMMANDS, treeMenuItems } from './contextmenu';

describe('treeMenuItems', () => {
    it('opens with the row action and ends with the commands that rewrite it', () => {
        const items = treeMenuItems('', false, false);
        expect(items[0]).toEqual({ id: 'open', label: 'Open' });
        expect(items[1]).toEqual({ id: 'reveal', label: 'Reveal in Finder' });
        expect(items.at(-1)).toEqual({ id: 'trash', label: 'Delete' });
    });

    it('offers the three path spellings together', () => {
        const items = treeMenuItems('', false, false);
        const start = items.findIndex((item) => item.id === 'copy-name');
        expect(items.slice(start, start + 3)).toEqual([
            { id: 'copy-name', label: 'Copy Name' },
            { id: 'copy-path', label: 'Copy Path' },
            { id: 'copy-relative-path', label: 'Copy Relative Path' },
        ]);
    });

    it('groups the rewriting commands after the clipboard ones', () => {
        const ids = treeMenuItems('', false, false).map((item) => item.id);
        expect(ids.indexOf('paste')).toBeLessThan(ids.indexOf('rename'));
        expect(ids.indexOf('rename')).toBeLessThan(ids.indexOf('duplicate'));
        expect(ids.indexOf('duplicate')).toBeLessThan(ids.indexOf('move'));
        expect(ids.indexOf('move')).toBeLessThan(ids.indexOf('trash'));
    });

    it('gives every command a unique id, so the menu keys are stable', () => {
        const ids = treeMenuItems('/tmp/file.ts', false, false)
            .map((item) => item.id)
            .filter((id) => id !== '');
        expect(new Set(ids).size).toBe(ids.length);
    });

    it('disables Paste until something is cut or copied', () => {
        expect(treeMenuItems('', false, false).find((item) => item.id === 'paste')?.disabled).toBe(true);
        expect(treeMenuItems('/tmp/file.ts', false, false).find((item) => item.id === 'paste')?.disabled).toBe(
            false,
        );
    });

    it('disables only the commands that move or delete a sidebar root', () => {
        const plain = treeMenuItems('', false, true);
        const root = treeMenuItems('', true, true);
        expect(root.slice(0, plain.length).map((item) => item.id)).toEqual(
            plain.map((item) => item.id),
        );
        expect(root.at(-1)).toEqual({ id: 'remove-folder', label: 'Remove Folder from Sidebar' });
        for (const item of root) {
            const locked = ROOT_LOCKED_COMMANDS.includes(item.id);
            expect(item.disabled === true).toBe(locked || item.id === 'paste');
        }
    });

    it('offers Remove Folder from Sidebar only for a root', () => {
        const ids = treeMenuItems('', false, false).map((item) => item.id);
        expect(ids).not.toContain('remove-folder');
    });

    it('offers Find in Folder beside Reveal in Finder, for folders only', () => {
        expect(treeMenuItems('', false, true)[2]).toEqual({
            id: 'find-in-folder',
            label: 'Find in Folder…',
        });
        expect(treeMenuItems('', true, true).map((item) => item.id)).toContain('find-in-folder');
        expect(treeMenuItems('', false, false).map((item) => item.id)).not.toContain(
            'find-in-folder',
        );
    });
});
