<script lang="ts">
    import FolderTree from './FolderTree.svelte';
    import { folderName } from './sidebar';

    interface Props {
        folders: string[];
        selectedPath?: string;
        busy?: boolean;
        onAddFolder: () => void;
        onRemoveFolder: (path: string) => void;
        onOpenFile: (path: string) => void;
    }

    let {
        folders,
        selectedPath = '',
        busy = false,
        onAddFolder,
        onRemoveFolder,
        onOpenFile,
    }: Props = $props();
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
        <ul class="sidebar__roots">
            {#each folders as folder (folder)}
                <FolderTree
                    path={folder}
                    name={folderName(folder)}
                    {selectedPath}
                    expanded={true}
                    removable={true}
                    {onOpenFile}
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
        flex: 0 0 auto;
        width: 260px;
        min-width: 160px;
        max-width: 40vw;
        background: var(--bg-sidebar);
        border-right: 1px solid var(--border);
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
        font-size: 11px;
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
