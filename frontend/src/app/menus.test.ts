import { describe, expect, it } from 'vitest';
import type { OpenDocument } from '../document';
import { treeMenuItems } from '../contextmenu';
import {
    contextMenuItems,
    contextMenuRoute,
    documentTabMenuItems,
    type MenuTarget,
} from './menus';

function tab(name: string, options: { path?: string; deleted?: boolean } = {}): OpenDocument {
    const path = options.path ?? `/repo/${name}`;
    return {
        id: path,
        path,
        realPath: path,
        name,
        kind: 'text',
        content: '',
        src: '',
        modified: 0,
        dirty: false,
        bom: false,
        eol: 'lf',
        invalidUTF8: false,
        deleted: options.deleted ?? false,
    };
}

function item(items: ReturnType<typeof documentTabMenuItems>, id: string) {
    return items.find((entry) => entry.id === id);
}

describe('contextMenuRoute', () => {
    const treeTarget: MenuTarget = { path: '/repo', isDir: true };

    it('drops a menu whose target is gone', () => {
        expect(contextMenuRoute('remove-folder', null, ['/repo'])).toBeNull();
    });

    it('sends Remove Folder from Sidebar for a root to the workspace', () => {
        expect(contextMenuRoute('remove-folder', treeTarget, ['/repo'])).toEqual({
            kind: 'remove-folder',
            path: '/repo',
        });
    });

    it('never removes a folder that is not a sidebar root', () => {
        expect(contextMenuRoute('remove-folder', treeTarget, ['/other'])).toBeNull();
    });

    it('sends every other tree item to the file commands', () => {
        expect(contextMenuRoute('trash', treeTarget, ['/repo'])).toEqual({
            kind: 'tree',
            id: 'trash',
            path: '/repo',
            isDir: true,
        });
    });

    it('sends a tab item to the tab it was opened on', () => {
        const target: MenuTarget = { path: '/repo/a.ts', isDir: false, tabId: 'tab-a' };

        expect(contextMenuRoute('reveal', target, ['/repo'])).toEqual({
            kind: 'tab',
            id: 'reveal',
            tabId: 'tab-a',
        });
    });
});

describe('documentTabMenuItems', () => {
    it('offers the tab commands, then the group closes, then Reveal, then the file commands, then Move', () => {
        const tabs = [tab('a.ts'), tab('b.ts')];
        const ids = documentTabMenuItems(tabs[0], tabs, false)
            .map((entry) => entry.id)
            .filter((id) => id !== '');

        expect(ids).toEqual([
            'close-tab',
            'close-other-tabs',
            'close-right-tabs',
            'close-deleted-tabs',
            'close-all-tabs',
            'reveal',
            'rename',
            'move',
            'duplicate',
            'trash',
            'move-tab-new-window',
        ]);
    });

    it('offers the file commands only for a tab with a file on disk', () => {
        const untitled: OpenDocument = {
            ...tab('Untitled-1'),
            id: 'untitled:Untitled-1',
            path: '',
            realPath: '',
        };
        const diff: OpenDocument = {
            ...tab('a.ts'),
            id: 'diff:/repo/a.ts',
            path: '',
            realPath: '',
            kind: 'diff',
            sourcePath: '/repo/a.ts',
        };

        for (const target of [untitled, diff]) {
            const ids = documentTabMenuItems(target, [target], false).map((entry) => entry.id);
            expect(ids).not.toContain('rename');
            expect(ids).not.toContain('trash');
        }
    });

    it('disables the file commands for the settings and keybindings files', () => {
        const settings = tab('settings.json', { path: '/config/settings.json' });
        const keys = tab('keybindings.json', { path: '/config/keybindings.json' });
        const config = ['/config/settings.json', '/config/keybindings.json'];

        for (const target of [settings, keys]) {
            const items = documentTabMenuItems(target, [target], false, config);
            for (const id of ['rename', 'move', 'duplicate', 'trash']) {
                expect(item(items, id)?.disabled).toBe(true);
            }
        }
        expect(item(documentTabMenuItems(tab('a.ts'), [tab('a.ts')], false, config), 'rename'))
            .toEqual({ id: 'rename', label: 'Rename…', disabled: false });
    });

    it('disables the file commands for a deleted file', () => {
        const deleted = tab('a.ts', { deleted: true });
        const items = documentTabMenuItems(deleted, [deleted], false);

        for (const id of ['rename', 'move', 'duplicate', 'trash']) {
            expect(item(items, id)?.disabled).toBe(true);
        }
    });

    it('disables Close Other Tabs for a lone tab', () => {
        const tabs = [tab('a.ts')];
        const items = documentTabMenuItems(tabs[0], tabs, false);

        expect(item(items, 'close-other-tabs')?.disabled).toBe(true);
        expect(item(items, 'close-all-tabs')?.disabled).toBe(false);
    });

    it('keeps Close Other Tabs enabled while the Find in Files tab is open', () => {
        const tabs = [tab('a.ts')];
        const items = documentTabMenuItems(tabs[0], tabs, true);

        expect(item(items, 'close-other-tabs')?.disabled).toBe(false);
        expect(item(items, 'close-all-tabs')?.disabled).toBe(false);
    });

    it('disables Close Tabs to Right on the last tab', () => {
        const tabs = [tab('a.ts'), tab('b.ts')];

        expect(item(documentTabMenuItems(tabs[1], tabs, false), 'close-right-tabs')?.disabled).toBe(
            true,
        );
        expect(item(documentTabMenuItems(tabs[0], tabs, false), 'close-right-tabs')?.disabled).toBe(
            false,
        );
    });

    it('keeps Close Tabs to Right enabled on the last tab when results are open', () => {
        const tabs = [tab('a.ts'), tab('b.ts')];
        const items = documentTabMenuItems(tabs[1], tabs, true);

        expect(item(items, 'close-right-tabs')?.disabled).toBe(false);
    });

    it('disables Close Tabs with Deleted Files when no tab lost its file', () => {
        const tabs = [tab('a.ts'), tab('b.ts')];

        expect(item(documentTabMenuItems(tabs[0], tabs, false), 'close-deleted-tabs')?.disabled).toBe(
            true,
        );
        const withDeleted = [tab('a.ts'), tab('gone.ts', { deleted: true })];
        expect(
            item(documentTabMenuItems(withDeleted[0], withDeleted, false), 'close-deleted-tabs')
                ?.disabled,
        ).toBe(false);
    });

    it('offers Reveal only for a tab with a file behind it', () => {
        const untitled: OpenDocument = {
            ...tab('Untitled-1'),
            id: 'untitled:Untitled-1',
            path: '',
            realPath: '',
        };
        const tabs = [untitled];

        expect(item(documentTabMenuItems(untitled, tabs, false), 'reveal')).toBeUndefined();
        expect(item(documentTabMenuItems(tab('a.ts'), [tab('a.ts')], false), 'reveal')).toEqual({
            id: 'reveal',
            label: 'Reveal in Finder',
        });
    });

    it('reveals the source file of a diff tab', () => {
        const diff: OpenDocument = {
            ...tab('a.ts'),
            id: 'diff:/repo/a.ts',
            path: '',
            realPath: '',
            kind: 'diff',
            sourcePath: '/repo/a.ts',
        };

        expect(item(documentTabMenuItems(diff, [diff], false), 'reveal')).toBeDefined();
    });
});

describe('contextMenuItems', () => {
    const tabs = [tab('a.ts')];
    const treeTarget: MenuTarget = { path: '/repo', isDir: true };

    it('shows nothing without a target', () => {
        expect(contextMenuItems(null, tabs, '', false, [])).toEqual([]);
    });

    it('offers the tree menu for a tree row', () => {
        expect(contextMenuItems(treeTarget, tabs, '', false, [])).toEqual(
            treeMenuItems('', false, true),
        );
    });

    it('disables moving or deleting a sidebar root', () => {
        const items = contextMenuItems(treeTarget, tabs, '', false, ['/repo']);

        for (const id of ['cut', 'rename', 'duplicate', 'move', 'trash']) {
            expect(item(items, id)?.disabled).toBe(true);
        }
        expect(item(items, 'new-file')?.disabled).toBeUndefined();
        expect(item(items, 'remove-folder')).toEqual({
            id: 'remove-folder',
            label: 'Remove Folder from Sidebar',
        });
    });

    it('treats a file with a root path spelling as a plain row', () => {
        const items = contextMenuItems({ path: '/repo', isDir: false }, tabs, '', false, ['/repo']);

        expect(item(items, 'trash')?.disabled).toBeUndefined();
    });

    it('enables Paste when a tree row has something cut', () => {
        const items = contextMenuItems(treeTarget, tabs, '/repo/a.ts', false, []);

        expect(item(items, 'paste')?.disabled).toBe(false);
    });

    it('offers the tab menu for a tab row', () => {
        const target: MenuTarget = { path: '/repo/a.ts', isDir: false, tabId: tabs[0].id };

        expect(contextMenuItems(target, tabs, '', false, [])).toEqual(
            documentTabMenuItems(tabs[0], tabs, false),
        );
    });

    it('shows nothing for a tab that has closed', () => {
        const target: MenuTarget = { path: '/repo/a.ts', isDir: false, tabId: 'gone' };

        expect(contextMenuItems(target, tabs, '', false, [])).toEqual([]);
    });

    it('lets an open Find in Files tab keep the group closes enabled', () => {
        const target: MenuTarget = { path: '/repo/a.ts', isDir: false, tabId: tabs[0].id };
        const items = contextMenuItems(target, tabs, '', true, []);

        expect(item(items, 'close-other-tabs')?.disabled).toBe(false);
    });
});
