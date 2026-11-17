import { describe, expect, it } from 'vitest';
import { treeMenuItems } from './contextmenu';

describe('treeMenuItems', () => {
    it('opens with the row action and ends with the commands that rewrite it', () => {
        const items = treeMenuItems('');
        expect(items[0]).toEqual({ id: 'open', label: 'Open' });
        expect(items[1]).toEqual({ id: 'reveal', label: 'Reveal in Finder' });
        expect(items.at(-1)).toEqual({ id: 'trash', label: 'Move to Trash' });
    });

    it('offers the three path spellings together', () => {
        const items = treeMenuItems('');
        const start = items.findIndex((item) => item.id === 'copy-name');
        expect(items.slice(start, start + 3)).toEqual([
            { id: 'copy-name', label: 'Copy Name' },
            { id: 'copy-path', label: 'Copy Path' },
            { id: 'copy-relative-path', label: 'Copy Relative Path' },
        ]);
    });

    it('groups the rewriting commands after the clipboard ones', () => {
        const ids = treeMenuItems('').map((item) => item.id);
        expect(ids.indexOf('paste')).toBeLessThan(ids.indexOf('rename'));
        expect(ids.indexOf('rename')).toBeLessThan(ids.indexOf('duplicate'));
        expect(ids.indexOf('duplicate')).toBeLessThan(ids.indexOf('move'));
        expect(ids.indexOf('move')).toBeLessThan(ids.indexOf('trash'));
    });

    it('gives every command a unique id, so the menu keys are stable', () => {
        const ids = treeMenuItems('/tmp/file.ts')
            .map((item) => item.id)
            .filter((id) => id !== '');
        expect(new Set(ids).size).toBe(ids.length);
    });

    it('disables Paste until something is cut or copied', () => {
        expect(treeMenuItems('').find((item) => item.id === 'paste')?.disabled).toBe(true);
        expect(treeMenuItems('/tmp/file.ts').find((item) => item.id === 'paste')?.disabled).toBe(
            false,
        );
    });
});
