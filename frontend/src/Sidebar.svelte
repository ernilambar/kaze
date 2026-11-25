<script lang="ts">
    import { onDestroy } from 'svelte';
    import FolderTree from './FolderTree.svelte';
    import type { GitStatusKind } from './git';
    import { folderName } from './sidebar';

    interface Props {
        folders: string[];
        expandedPaths: readonly string[];
        restoredExpansionRoots: readonly string[];
        selectedPath?: string;
        busy?: boolean;
        // refreshToken changes when the tree should reread what is on disk.
        refreshToken?: number;
        // busyPath is the entry a tree command is running on, so its row can say
        // that the command is still working.
        busyPath?: string;
        // statuses maps a path to its git state for the row's status mark.
        statuses?: Record<string, GitStatusKind>;
        // folderChanges maps a folder to the most significant change beneath
        // it, so a folder row can mark itself with the matching colour.
        folderChanges?: Map<string, GitStatusKind>;
        onAddFolder: () => void;
        onRemoveFolder: (path: string) => void;
        onOpenFile: (path: string) => void;
        // onContextMenu opens the row menu for a file or folder.
        onContextMenu?: (
            event: MouseEvent,
            path: string,
            options: { isDir?: boolean },
        ) => void;
        // onEmptyAreaMenu opens the same menu for the folder behind an empty
        // area of the tree.
        onEmptyAreaMenu?: (event: MouseEvent, folder: string) => void;
        // onMovePath moves an entry into a folder, which is what dropping it on
        // that folder row does.
        onMovePath?: (path: string, targetDir: string) => void;
        // onExpandedChange reports which folders are open, so the app can watch
        // them for external changes.
        onExpandedChange?: (path: string, open: boolean) => void;
        iconStyle?: 'colorful' | 'monochrome';
    }

    let {
        folders,
        expandedPaths,
        restoredExpansionRoots,
        selectedPath = '',
        busy = false,
        refreshToken = 0,
        busyPath = '',
        statuses = {},
        folderChanges = new Map(),
        onAddFolder,
        onRemoveFolder,
        onOpenFile,
        onContextMenu,
        onEmptyAreaMenu,
        onMovePath,
        onExpandedChange,
        iconStyle = 'colorful',
    }: Props = $props();

    // The tree's scrollbar is only worth showing while the user is scrolling;
    // a short tail keeps it from flickering between wheel ticks.
    const SCROLLBAR_TAIL_MS = 700;
    let scrolling = $state(false);
    let hideTimer: ReturnType<typeof setTimeout> | undefined;

    function onRootsScroll() {
        scrolling = true;
        clearTimeout(hideTimer);
        hideTimer = setTimeout(() => {
            scrolling = false;
        }, SCROLLBAR_TAIL_MS);
    }

    onDestroy(() => clearTimeout(hideTimer));
</script>

<aside class="sidebar">
    <div class="sidebar__header">
        <span class="sidebar__title">Explorer</span>
        <button
            class="sidebar__action"
            disabled={busy}
            onclick={onAddFolder}
            title="Add Folder to Sidebar"
        >
            <svg aria-hidden="true" height="12" viewBox="0 0 12 12" width="12">
                <path d="M6 1 L6 11 M1 6 L11 6" fill="none" stroke="currentColor" stroke-width="1.5" />
            </svg>
        </button>
    </div>

    {#if folders.length === 0}
        <div class="sidebar__empty">
            <p>No folders yet.</p>
            <button class="sidebar__add" disabled={busy} onclick={onAddFolder}>Add Folder…</button>
        </div>
    {:else}
        <ul class="sidebar__roots" class:sidebar__roots--scrolling={scrolling} onscroll={onRootsScroll}>
            {#each folders as folder (folder)}
                <FolderTree
                    path={folder}
                    name={folderName(folder)}
                    {selectedPath}
                    {refreshToken}
                    {busyPath}
                    {statuses}
                    {folderChanges}
                    {expandedPaths}
                    {restoredExpansionRoots}
                    removable={true}
                    {onOpenFile}
                    {onContextMenu}
                    {onEmptyAreaMenu}
                    {onMovePath}
                    {onExpandedChange}
                    {iconStyle}
                    onRemove={onRemoveFolder}
                />
            {/each}
        </ul>
    {/if}
</aside>

<style>
    .sidebar {
        display: flex;
        flex-direction: column;
        flex: 1 1 auto;
        min-width: 0;
        background: var(--bg-sidebar);
        border-right: 1px solid var(--border);
        -webkit-user-select: none;
        user-select: none;
        overflow: hidden;
    }

    .sidebar__header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex: 0 0 auto;
        height: 32px;
        padding: 0 6px 0 12px;
    }

    .sidebar__title {
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
        font-weight: 600;
        letter-spacing: 0.06em;
        text-transform: uppercase;
    }

    .sidebar__action {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 22px;
        height: 22px;
        padding: 0;
        border: none;
        border-radius: 3px;
        background: none;
        color: var(--fg-muted);
        cursor: pointer;
    }

    .sidebar__action:hover:enabled {
        background: var(--bg-hover);
        color: var(--fg);
    }

    .sidebar__action:disabled {
        cursor: default;
        opacity: 0.5;
    }

    .sidebar__roots {
        flex: 1 1 auto;
        margin: 0;
        padding: 0 0 8px;
        list-style: none;
        overflow: auto;
        scrollbar-width: thin;
        scrollbar-color: transparent transparent;
    }

    /* The native scrollbar is always drawn and loud, so the tree gets a slim
       one whose thumb only shows while the list is scrolling. The track stays
       reserved, so rows don't shift. */
    .sidebar__roots::-webkit-scrollbar {
        width: 10px;
    }

    .sidebar__roots::-webkit-scrollbar-track {
        background: transparent;
    }

    .sidebar__roots::-webkit-scrollbar-thumb {
        background: transparent;
        border: 2px solid transparent;
        border-radius: 6px;
        background-clip: padding-box;
    }

    .sidebar__roots--scrolling {
        scrollbar-color: var(--border) transparent;
    }

    .sidebar__roots--scrolling::-webkit-scrollbar-thumb {
        background: var(--border);
        background-clip: padding-box;
    }

    .sidebar__roots--scrolling::-webkit-scrollbar-thumb:hover {
        background: var(--fg-muted);
        background-clip: padding-box;
    }

    .sidebar__empty {
        display: flex;
        flex-direction: column;
        align-items: flex-start;
        gap: 8px;
        padding: 4px 12px 12px;
        color: var(--fg-muted);
    }

    .sidebar__empty p {
        margin: 0;
    }

    .sidebar__add {
        padding: 4px 10px;
        border: 1px solid var(--border);
        border-radius: 4px;
        background: var(--bg-raised);
        color: var(--fg);
        font: inherit;
        cursor: pointer;
    }

    .sidebar__add:hover:enabled {
        background: var(--bg-hover);
    }

    .sidebar__add:disabled {
        cursor: default;
        opacity: 0.5;
    }
</style>
