<script lang="ts">
    import { onMount, untrack } from 'svelte';
    import Self from './FolderTree.svelte';
    import { errorMessage, isWithin, readFolder, type FolderEntry } from './sidebar';

    interface Props {
        path: string;
        name: string;
        depth?: number;
        selectedPath?: string;
        expanded?: boolean;
        removable?: boolean;
        onOpenFile: (path: string) => void;
        onRemove?: (path: string) => void;
    }

    let {
        path,
        name,
        depth = 0,
        selectedPath = '',
        expanded = false,
        removable = false,
        onOpenFile,
        onRemove,
    }: Props = $props();

    // `expanded` is only the initial state: a root starts open, and after that
    // the user's toggles own it.
    // svelte-ignore state_referenced_locally
    let open = $state(expanded);
    let entries = $state<FolderEntry[]>([]);
    let error = $state('');
    let loaded = $state(false);
    let loading = $state(false);

    // Children are read when the folder is first shown, so adding a large
    // folder costs one directory read rather than a walk over the whole tree.
    // Re-expanding forces a refresh, because files outside Kova come and go.
    async function ensureLoaded(force = false): Promise<void> {
        if (loading || (loaded && !force)) {
            return;
        }
        loading = true;
        try {
            entries = await readFolder(path);
            loaded = true;
            error = '';
        } catch (failure) {
            error = errorMessage(failure);
        } finally {
            loading = false;
        }
    }

    function toggle(): void {
        open = !open;
        if (open) {
            void ensureLoaded(true);
        }
    }

    // Opening a file from the command line should show where it lives, so a
    // node on the selected file's path expands itself and loads its children.
    // Only the selection is tracked: reading `open` here would re-expand a
    // folder the user just collapsed.
    $effect(() => {
        const selected = selectedPath;
        reveal(selected);
    });

    function reveal(selected: string): void {
        untrack(() => {
            if (selected === '' || !isWithin(path, selected)) {
                return;
            }
            open = true;
            void ensureLoaded();
        });
    }

    onMount(() => {
        if (open) {
            void ensureLoaded();
        }
    });
</script>

<li class="node">
    <div class="row" style:padding-left={`${8 + depth * 14}px`}>
        <button class="row__main" onclick={toggle} title={path}>
            <span class="row__chevron" class:row__chevron--open={open}>
                <svg aria-hidden="true" height="10" viewBox="0 0 10 10" width="10">
                    <path d="M3 1 L7 5 L3 9" fill="none" stroke="currentColor" stroke-width="1.5" />
                </svg>
            </span>
            <span class="row__name">{name}</span>
        </button>
        {#if removable && onRemove !== undefined}
            <button class="row__remove" onclick={() => onRemove?.(path)} title={`Remove ${name}`}>
                <svg aria-hidden="true" height="10" viewBox="0 0 10 10" width="10">
                    <path d="M1 1 L9 9 M9 1 L1 9" fill="none" stroke="currentColor" stroke-width="1.5" />
                </svg>
            </button>
        {/if}
    </div>

    {#if open}
        {#if error !== ''}
            <p class="node__error" style:padding-left={`${22 + depth * 14}px`}>{error}</p>
        {:else if loaded && entries.length === 0}
            <p class="node__empty" style:padding-left={`${22 + depth * 14}px`}>Empty</p>
        {:else}
            <ul class="node__children">
                {#each entries as entry (entry.path)}
                    {#if entry.isDir}
                        <Self depth={depth + 1} name={entry.name} path={entry.path} {selectedPath} {onOpenFile} />
                    {:else}
                        <li class="node">
                            <div
                                class="row"
                                class:row--selected={selectedPath === entry.path}
                                style:padding-left={`${8 + (depth + 1) * 14}px`}
                            >
                                <button class="row__main" onclick={() => onOpenFile(entry.path)} title={entry.path}>
                                    <span class="row__spacer"></span>
                                    <span class="row__name">{entry.name}</span>
                                </button>
                            </div>
                        </li>
                    {/if}
                {/each}
            </ul>
        {/if}
    {/if}
</li>

<style>
    .node__children {
        margin: 0;
        padding: 0;
        list-style: none;
    }

    .row {
        display: flex;
        align-items: center;
        min-height: 22px;
    }

    .row:hover {
        background: var(--bg-raised);
    }

    .row--selected {
        background: var(--bg-selected);
    }

    .row__main {
        display: flex;
        align-items: center;
        gap: 4px;
        flex: 1 1 auto;
        min-width: 0;
        padding: 2px 4px;
        border: none;
        background: none;
        color: var(--fg);
        font: inherit;
        text-align: left;
        cursor: pointer;
    }

    .row__chevron {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        flex: 0 0 auto;
        width: 10px;
        color: var(--fg-muted);
        transition: transform 80ms ease-out;
    }

    .row__chevron--open {
        transform: rotate(90deg);
    }

    .row__spacer {
        flex: 0 0 auto;
        width: 10px;
    }

    .row__name {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .row__remove {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        flex: 0 0 auto;
        width: 20px;
        height: 20px;
        margin-right: 4px;
        padding: 0;
        border: none;
        border-radius: 3px;
        background: none;
        color: var(--fg-muted);
        cursor: pointer;
        opacity: 0;
    }

    .row:hover .row__remove,
    .row__remove:focus-visible {
        opacity: 1;
    }

    .row__remove:hover {
        background: var(--bg-hover);
        color: var(--fg);
    }

    .node__error {
        margin: 0;
        padding-top: 2px;
        padding-bottom: 4px;
        color: var(--fg-danger);
        font-size: 12px;
    }

    .node__empty {
        margin: 0;
        padding-top: 2px;
        padding-bottom: 4px;
        color: var(--fg-muted);
        font-size: 12px;
        font-style: italic;
    }
</style>
