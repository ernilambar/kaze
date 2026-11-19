<script lang="ts">
    import { onMount, untrack } from 'svelte';
    import Self from './FolderTree.svelte';
    import {
        DRAG_TYPE,
        draggingPath,
        dropSource,
        setDraggingPath,
    } from './filetree';
    import { fileIcon } from './fileIcons';
    import { statusLabel, statusMark, type GitStatusKind } from './git';
    import { errorMessage, isWithin, readFolder, type FolderEntry } from './sidebar';

    interface Props {
        path: string;
        // root is the sidebar folder this row belongs to; files.exclude
        // patterns match paths relative to it. A root row leaves it unset.
        root?: string;
        name: string;
        depth?: number;
        selectedPath?: string;
        refreshToken?: number;
        // busyPath is the entry a tree command is running on, so its row can say
        // that the command is still working.
        busyPath?: string;
        statuses?: Record<string, GitStatusKind>;
        // folderChanges is the set of folders with changed descendants, so a
        // collapsed folder still shows that it hides work.
        folderChanges?: Set<string>;
        expanded?: boolean;
        removable?: boolean;
        onOpenFile: (path: string) => void;
        // onContextMenu opens the row menu for a file or folder.
        onContextMenu?: (
            event: MouseEvent,
            path: string,
            options: { isDir?: boolean },
        ) => void;
        onRemove?: (path: string) => void;
        onExpandedChange?: (path: string, open: boolean) => void;
        iconStyle?: 'colorful' | 'monochrome';
        // onMovePath moves path into targetDir. It is what a drop on a folder
        // row does, from the tree or from another app.
        onMovePath?: (path: string, targetDir: string) => void;
        // onEmptyAreaMenu opens the row menu for the folder behind the empty
        // space around a folder's children.
        onEmptyAreaMenu?: (event: MouseEvent, folder: string) => void;
    }

    let {
        path,
        root,
        name,
        depth = 0,
        selectedPath = '',
        refreshToken = 0,
        busyPath = '',
        statuses = {},
        folderChanges = new Set(),
        expanded = false,
        removable = false,
        onOpenFile,
        onContextMenu,
        onRemove,
        onExpandedChange,
        iconStyle = 'colorful',
        onMovePath,
        onEmptyAreaMenu,
    }: Props = $props();

    // `expanded` is only the initial state: a root starts open, and after that
    // the user's toggles own it.
    // svelte-ignore state_referenced_locally
    let open = $state(expanded);
    let entries = $state<FolderEntry[]>([]);
    // folderChanged marks a folder that hides a git change beneath it, so a
    // collapsed row still says so.
    let folderChanged = $derived(folderChanges.has(path));
    let error = $state('');
    let loaded = $state(false);
    let loading = $state(false);
    let reloadPending = false;

    // dropTarget marks the folder a drag is over, which is also the row that
    // draws the highlight. What is being dragged lives in filetree.ts, because a
    // drag spans several rows and each one is its own component.
    let dropTarget = $state(false);

    function onRowDragStart(event: DragEvent): void {
        startDrag(event, path);
    }

    // onFileDragStart drags a file row, so a file can be moved into any folder
    // row in the tree. The folder rows are the drop targets; a file row is not.
    function onFileDragStart(event: DragEvent, filePath: string): void {
        startDrag(event, filePath);
    }

    function startDrag(event: DragEvent, source: string): void {
        setDraggingPath(source);
        if (event.dataTransfer === null) {
            return;
        }
        event.dataTransfer.effectAllowed = 'move';
        event.dataTransfer.setData(DRAG_TYPE, source);
        event.dataTransfer.setData('text/plain', source);
    }

    function onRowDragEnd(): void {
        onDragFinished();
    }

    // onDragFinished clears the state a drag left behind. A drop clears it too,
    // because the browser does not always send dragend after a successful drop.
    function onDragFinished(): void {
        setDraggingPath('');
        dropTarget = false;
    }

    // canDropPath keeps a folder from being dropped into itself or into its own
    // subtree, which would detach it from the tree.
    function canDropPath(source: string): boolean {
        if (source === '' || source === path) {
            return false;
        }
        return !isWithin(source, path);
    }

    function onRowDragOver(event: DragEvent): void {
        const source = draggingPath() || dropSource(event.dataTransfer);
        if (!canDropPath(source)) {
            return;
        }
        event.preventDefault();
        dropTarget = true;
        if (event.dataTransfer !== null) {
            event.dataTransfer.dropEffect = 'move';
        }
    }

    function onRowDragLeave(): void {
        dropTarget = false;
    }

    function onRowDrop(event: DragEvent): void {
        event.preventDefault();
        const source = dropSource(event.dataTransfer);
        onDragFinished();
        if (!canDropPath(source)) {
            return;
        }
        onMovePath?.(source, path);
    }

    // Children are read when the folder is first shown, so adding a large
    // folder costs one directory read rather than a walk over the whole tree.
    // Re-expanding forces a refresh, because files outside the app come and go.
    async function ensureLoaded(force = false): Promise<void> {
        if (loading) {
            // A forced refresh during a read may be for a change that read
            // missed; run it once the read finishes.
            reloadPending ||= force;
            return;
        }
        if (loaded && !force) {
            return;
        }
        loading = true;
        try {
            entries = await readFolder(path, root ?? path);
            loaded = true;
            error = '';
        } catch (failure) {
            error = errorMessage(failure);
        } finally {
            loading = false;
        }
        if (reloadPending) {
            reloadPending = false;
            await ensureLoaded(true);
        }
    }

    function toggle(): void {
        open = !open;
        onExpandedChange?.(path, open);
        if (open) {
            void ensureLoaded(true);
        }
    }

    // Refresh rereads the children of an open node. A token rather than a
    // boolean lets repeated refreshes, from the menu or from external changes,
    // each land.
    let lastRefreshToken = -1;
    $effect(() => {
        const token = refreshToken;
        if (token === lastRefreshToken) {
            return;
        }
        lastRefreshToken = token;
        untrack(() => {
            if (open) {
                void ensureLoaded(true);
            }
        });
    });

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
            // A folder opened to reveal a file is open like any other: it is
            // watched and saved in the session.
            if (!open) {
                open = true;
                onExpandedChange?.(path, true);
            }
            void ensureLoaded();
        });
    }

    // revealSelected scrolls the active row into view once the tree has
    // expanded to it, so a deep file does not stay below the fold.
    function revealSelected(
        node: HTMLElement,
        selected: boolean,
    ): { update(selected: boolean): void } {
        if (selected) {
            node.scrollIntoView({ block: 'nearest' });
        }
        return {
            update(next: boolean) {
                if (next) {
                    node.scrollIntoView({ block: 'nearest' });
                }
            },
        };
    }

    onMount(() => {
        onExpandedChange?.(path, open);
        if (open) {
            void ensureLoaded();
        }
        return () => onExpandedChange?.(path, false);
    });
</script>

<li class="node">
    <div
        class="row"
        role="presentation"
        class:row--drop={dropTarget}
        class:row--busy={busyPath === path}
        aria-busy={busyPath === path}
        style:padding-left={`${8 + depth * 14}px`}
        draggable={onMovePath !== undefined && depth > 0}
        ondragstart={onRowDragStart}
        ondragend={onRowDragEnd}
        ondragover={onRowDragOver}
        ondragleave={onRowDragLeave}
        ondrop={onRowDrop}
    >
        <button
            class="row__main"
            onclick={toggle}
            title={path}
            aria-label={folderChanged ? `${name} — has git changes` : undefined}
            oncontextmenu={(event) => onContextMenu?.(event, path, { isDir: true })}
        >
            <span class="row__chevron" class:row__chevron--open={open}>
                <svg aria-hidden="true" height="10" viewBox="0 0 10 10" width="10">
                    <path d="M3 1 L7 5 L3 9" fill="none" stroke="currentColor" stroke-width="1.5" />
                </svg>
            </span>
            <span class="row__icon row__icon--folder" aria-hidden="true">
                <svg height="15" viewBox="0 0 16 16" width="15">
                    <path
                        d="M2 4.25A1.25 1.25 0 0 1 3.25 3h3.1c.33 0 .65.13.88.37l1.06 1.06c.23.24.55.37.88.37h4.58A1.25 1.25 0 0 1 15 6.05v6.7A1.25 1.25 0 0 1 13.75 14H3.25A1.25 1.25 0 0 1 2 12.75z"
                        fill="currentColor"
                    />
                </svg>
            </span>
            <span class="row__name" class:row__name--changed={folderChanged}>{name}</span>
            {#if folderChanged}
                <span class="row__changed" aria-hidden="true"></span>
            {/if}
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
        <!-- The area below a row belongs to the folder that owns it, so a
             right-click on the space around its children offers the same menu
             the folder row does. -->
        <div
            class="node__area"
            role="presentation"
            oncontextmenu={(event) => onEmptyAreaMenu?.(event, path)}
        >
        {#if error !== ''}
            <p class="node__error" style:padding-left={`${22 + depth * 14}px`}>{error}</p>
        {:else if loaded && entries.length === 0}
            <p class="node__empty" style:padding-left={`${22 + depth * 14}px`}>Empty</p>
        {:else}
            <ul class="node__children">
                {#each entries as entry (entry.path)}
                    {#if entry.isDir}
                        <Self
                            depth={depth + 1}
                            name={entry.name}
                            path={entry.path}
                            root={root ?? path}
                            {selectedPath}
                            {refreshToken}
                            {busyPath}
                            {statuses}
                            {folderChanges}
                            {onOpenFile}
                            {onContextMenu}
                            {onExpandedChange}
                            {iconStyle}
                            {onMovePath}
                            {onEmptyAreaMenu}
                        />
                    {:else}
                        {@const icon = fileIcon(entry.name)}
                        {@const status = statuses[entry.path]}
                        <li class="node">
                            <div
                                class="row"
                                role="presentation"
                                class:row--selected={selectedPath === entry.path}
                                class:row--busy={busyPath === entry.path}
                                style:padding-left={`${8 + (depth + 1) * 14}px`}
                                draggable={onMovePath !== undefined}
                                ondragstart={(event) => onFileDragStart(event, entry.path)}
                                ondragend={onRowDragEnd}
                                use:revealSelected={selectedPath === entry.path}
                            >
                                <button
                                    class="row__main"
                                    onclick={() => onOpenFile(entry.path)}
                                    title={entry.path}
                                    oncontextmenu={(event) =>
                                        onContextMenu?.(event, entry.path, {
                                            isDir: false,
                                        })}
                                >
                                    <span class="row__spacer"></span>
                                    <span
                                        class="row__icon"
                                        aria-hidden="true"
                                        style:color={iconStyle === 'monochrome' ? 'var(--fg-muted)' : icon.color}
                                        style:font-family={icon.family}
                                        style:font-size={`${icon.size}px`}
                                        style:top={`${icon.top}px`}
                                        style:left={`${icon.left}px`}
                                    >{icon.glyph}</span>
                                    <span class="row__name">{entry.name}</span>
                                    {#if status !== undefined}
                                        <span
                                            class="row__status row__status--{status}"
                                            title={statusLabel(status)}
                                            aria-label={statusLabel(status)}
                                        >{statusMark(status)}</span>
                                    {/if}
                                </button>
                            </div>
                        </li>
                    {/if}
                {/each}
            </ul>
        {/if}
        </div>
    {/if}
</li>

<style>
    /* The area holds a folder's children and the empty space around them; only
       the list itself is styled, so the space stays clickable but invisible. */
    .node__area {
        display: block;
    }

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

    /* The entry a command is working on, so a slow copy does not look stuck. */
    .row--busy {
        opacity: 0.5;
        cursor: progress;
    }

    /* The folder a drag would drop into, while the pointer is over it. */
    .row--drop {
        background: var(--bg-selected);
        outline: 1px solid var(--fg-muted);
        outline-offset: -1px;
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

    .row__icon {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        flex: 0 0 auto;
        width: 16px;
        height: 16px;
        position: relative;
        line-height: 1;
        -webkit-font-smoothing: antialiased;
    }

    .row__icon--folder {
        color: var(--fg-muted);
    }

    .row:hover .row__icon--folder {
        color: var(--fg);
    }

    .row__name {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    /* A folder with a git change beneath it: the name and a dot both take the
       warning colour, so a collapsed folder still shows where the work is. */
    .row__name--changed {
        color: var(--git-conflicted);
    }

    .row__changed {
        flex: 0 0 auto;
        margin-left: auto;
        width: 7px;
        height: 7px;
        border-radius: 50%;
        background: var(--git-conflicted);
    }

    .row__status {
        flex: 0 0 auto;
        margin-left: auto;
        padding: 0 4px;
        font-family: Menlo, monospace;
        font-size: var(--ui-font-small);
        font-weight: 600;
    }

    .row__status--modified {
        color: var(--git-modified);
    }

    .row__status--added,
    .row__status--untracked {
        color: var(--git-added);
    }

    .row__status--deleted {
        color: var(--git-deleted);
    }

    .row__status--conflicted {
        color: var(--git-conflicted);
    }

    .row__status--renamed,
    .row__status--copied {
        color: var(--fg-muted);
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
        font-size: calc(var(--ui-font-size) - 1px);
    }

    .node__empty {
        margin: 0;
        padding-top: 2px;
        padding-bottom: 4px;
        color: var(--fg-muted);
        font-size: calc(var(--ui-font-size) - 1px);
        font-style: italic;
    }
</style>
