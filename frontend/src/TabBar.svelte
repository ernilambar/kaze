<script lang="ts">
    import type { OpenDocument } from './document';
    import { fileIcon } from './fileIcons';

    interface Props {
        tabs: OpenDocument[];
        activeId: string;
        // The Find in Files results view is a tab too, but it has no document
        // behind it, so it is rendered from these props instead of the list.
        searchOpen?: boolean;
        searchActive?: boolean;
        onSelect: (id: string) => void;
        onClose: (id: string) => void;
        // onContextMenu opens the tab's row menu; the tab carries the document
        // so the menu can act on the file under the pointer.
        onContextMenu?: (event: MouseEvent, tab: OpenDocument) => void;
        // onReorder drops the dragged tab before or after the target tab.
        onReorder: (id: string, targetId: string, placeAfter: boolean) => void;
        onSelectSearch?: () => void;
        onCloseSearch?: () => void;
        // showIcons draws the file-type glyph next to each label.
        showIcons?: boolean;
        iconStyle?: 'colorful' | 'monochrome';
    }

    let {
        tabs,
        activeId,
        searchOpen = false,
        searchActive = false,
        onSelect,
        onClose,
        onContextMenu,
        onReorder,
        onSelectSearch,
        onCloseSearch,
        showIcons = true,
        iconStyle = 'colorful',
    }: Props = $props();

    function label(tab: OpenDocument): string {
        if (tab.kind === 'diff') {
            return tab.sourcePath ?? tab.name;
        }
        return tab.path !== '' ? tab.path : `${tab.name} (unsaved)`;
    }

    // draggedId is the tab being dragged; dropTarget is the tab whose left or
    // right edge shows the insertion line. Both clear on dragend, so an aborted
    // drag cannot leave a stale indicator behind.
    let draggedId = $state<string | null>(null);
    let dropTarget = $state<{ id: string; after: boolean } | null>(null);

    // dropAfter reads which half of the tab the pointer is over, so a drop lands
    // on the side the insertion line promised.
    function dropAfter(event: DragEvent): boolean {
        const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
        return event.clientX > rect.left + rect.width / 2;
    }

    function startDrag(event: DragEvent, id: string): void {
        draggedId = id;
        if (event.dataTransfer !== null) {
            event.dataTransfer.effectAllowed = 'move';
            // WebKit refuses to start a drag without payload data.
            event.dataTransfer.setData('text/plain', id);
        }
    }

    function dragOver(event: DragEvent, id: string): void {
        if (draggedId === null) {
            return;
        }
        event.preventDefault();
        if (event.dataTransfer !== null) {
            event.dataTransfer.dropEffect = 'move';
        }
        dropTarget = draggedId === id ? null : { id, after: dropAfter(event) };
    }

    function dropOn(event: DragEvent, id: string): void {
        event.preventDefault();
        event.stopPropagation();
        const dragged = draggedId;
        draggedId = null;
        dropTarget = null;
        if (dragged !== null && dragged !== id) {
            onReorder(dragged, id, dropAfter(event));
        }
    }

    // The strip's empty space and the Search Results tab both mean "after the
    // last document tab", so a tab can be dragged to the far end.
    function dragOverEnd(event: DragEvent): void {
        const last = tabs[tabs.length - 1];
        if (draggedId === null || last === undefined || draggedId === last.id) {
            return;
        }
        event.preventDefault();
        if (event.dataTransfer !== null) {
            event.dataTransfer.dropEffect = 'move';
        }
        dropTarget = { id: last.id, after: true };
    }

    // The strip's empty space only accepts the drop; the indicator stays on the
    // last tab, where dragOverEnd put it. Events from a tab bubble here too, so
    // only a hit on the strip itself moves the indicator.
    function dragOverStrip(event: DragEvent): void {
        if (draggedId === null) {
            return;
        }
        event.preventDefault();
        if (event.target === event.currentTarget) {
            dragOverEnd(event);
        }
    }

    function dropAtEnd(event: DragEvent): void {
        const last = tabs[tabs.length - 1];
        const dragged = draggedId;
        draggedId = null;
        dropTarget = null;
        if (dragged === null || last === undefined || dragged === last.id) {
            return;
        }
        event.preventDefault();
        event.stopPropagation();
        onReorder(dragged, last.id, true);
    }

    function endDrag(): void {
        draggedId = null;
        dropTarget = null;
    }

    // dragleave fires when the pointer crosses into a child too; only an exit
    // from the whole strip drops the indicator.
    function leaveStrip(event: DragEvent): void {
        const next = event.relatedTarget as Node | null;
        if (next === null || !(event.currentTarget as HTMLElement).contains(next)) {
            dropTarget = null;
        }
    }
</script>

{#if tabs.length > 0 || searchOpen}
    <div
        class="tabs"
        role="tablist"
        tabindex="-1"
        aria-label="Open files"
        ondragleave={leaveStrip}
        ondragover={dragOverStrip}
        ondrop={dropAtEnd}
    >
        {#each tabs as tab (tab.id)}
            {@const icon = fileIcon(tab.name)}
            <div
                class="tab"
                role="presentation"
                class:tab--active={tab.id === activeId}
                class:tab--deleted={tab.deleted}
                class:tab--dragging={draggedId === tab.id}
                class:tab--drop-before={dropTarget?.id === tab.id && !dropTarget.after}
                class:tab--drop-after={dropTarget?.id === tab.id && dropTarget.after}
                draggable="true"
                ondragstart={(event) => startDrag(event, tab.id)}
                ondragover={(event) => dragOver(event, tab.id)}
                ondrop={(event) => dropOn(event, tab.id)}
                ondragend={endDrag}
                oncontextmenu={(event) => onContextMenu?.(event, tab)}
            >
                {#if showIcons}
                    <span
                        class="tab__icon"
                        aria-hidden="true"
                        style:color={iconStyle === 'monochrome' ? 'var(--fg-muted)' : icon.color}
                        style:font-family={icon.family}
                        style:font-size={`${icon.size}px`}
                        style:top={`${icon.top}px`}
                        style:left={`${icon.left}px`}
                    >{icon.glyph}</span>
                {/if}
                <button
                    class="tab__label"
                    role="tab"
                    aria-selected={tab.id === activeId}
                    title={tab.deleted ? `${label(tab)} (deleted)` : label(tab)}
                    onclick={() => onSelect(tab.id)}
                >{tab.name}</button>
                {#if tab.dirty}
                    <span class="tab__dot" role="img" aria-label="Unsaved changes" title="Unsaved changes"></span>
                {/if}
                <button
                    class="tab__close"
                    aria-label={`Close ${tab.name}`}
                    title={`Close ${tab.name}`}
                    onclick={() => onClose(tab.id)}
                >
                    <svg aria-hidden="true" height="10" viewBox="0 0 10 10" width="10">
                        <path d="M1.5 1.5 L8.5 8.5 M8.5 1.5 L1.5 8.5" fill="none" stroke="currentColor" stroke-width="1.4" />
                    </svg>
                </button>
            </div>
        {/each}
        {#if searchOpen}
            <div
                class="tab"
                role="presentation"
                class:tab--active={searchActive}
                ondragover={dragOverEnd}
                ondrop={dropAtEnd}
            >
                <button
                    class="tab__label"
                    role="tab"
                    aria-selected={searchActive}
                    title="Find in Files"
                    onclick={onSelectSearch}
                >Search Results</button>
                <button
                    class="tab__close"
                    aria-label="Close Search Results"
                    title="Close Search Results"
                    onclick={onCloseSearch}
                >
                    <svg aria-hidden="true" height="10" viewBox="0 0 10 10" width="10">
                        <path d="M1.5 1.5 L8.5 8.5 M8.5 1.5 L1.5 8.5" fill="none" stroke="currentColor" stroke-width="1.4" />
                    </svg>
                </button>
            </div>
        {/if}
    </div>
{/if}

<style>
    .tabs {
        display: flex;
        flex: 0 0 auto;
        height: 32px;
        background: var(--bg-raised);
        border-bottom: 1px solid var(--border);
        overflow-x: auto;
        overflow-y: hidden;
        scrollbar-width: none;
        user-select: none;
    }

    .tabs::-webkit-scrollbar {
        display: none;
    }

    .tab {
        display: flex;
        align-items: stretch;
        flex: 0 1 auto;
        min-width: 0;
        max-width: 220px;
        border-right: 1px solid var(--border);
        color: var(--fg-muted);
        background: var(--bg-raised);
    }

    .tab:hover {
        background: var(--bg-hover);
    }

    .tab--active {
        color: var(--fg);
        background: var(--bg);
        box-shadow: inset 0 1px 0 var(--fg-muted);
    }

    .tab--active:hover {
        background: var(--bg);
    }

    .tab--deleted .tab__label {
        text-decoration: line-through;
    }

    .tab--dragging {
        opacity: 0.5;
    }

    /* The insertion line sits on the edge the tab will be dropped against, so
       active tabs lose their top line only while a drag hovers them. */
    .tab--drop-before {
        box-shadow: inset 2px 0 0 var(--fg);
    }

    .tab--drop-after {
        box-shadow: inset -2px 0 0 var(--fg);
    }

    .tab__icon {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        flex: 0 0 auto;
        align-self: center;
        width: 16px;
        height: 16px;
        margin-left: 8px;
        position: relative;
        line-height: 1;
        -webkit-font-smoothing: antialiased;
    }

    .tab__icon + .tab__label {
        padding-left: 4px;
    }

    .tab__label {
        flex: 1 1 auto;
        min-width: 0;
        padding: 0 8px 0 12px;
        border: none;
        background: none;
        color: inherit;
        font: inherit;
        text-align: left;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        cursor: pointer;
    }

    .tab__label:focus-visible,
    .tab__close:focus-visible {
        outline: 1px solid var(--fg-muted);
        outline-offset: -2px;
    }

    /* A modified buffer shows a dot where the close button will appear, so a
       tab keeps one width whether it can be closed or only saved. */
    .tab__dot {
        flex: 0 0 auto;
        align-self: center;
        width: 7px;
        height: 7px;
        margin-right: 8px;
        border-radius: 50%;
        background: currentColor;
    }

    .tab:hover .tab__dot {
        display: none;
    }

    .tab__close {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        flex: 0 0 auto;
        width: 26px;
        padding: 0;
        border: none;
        background: none;
        color: inherit;
        cursor: pointer;
        opacity: 0;
    }

    .tab:hover .tab__close,
    .tab__close:focus-visible {
        opacity: 1;
    }

    .tab__close:hover {
        color: var(--fg);
    }
</style>
