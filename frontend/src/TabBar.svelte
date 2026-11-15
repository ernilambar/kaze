<script lang="ts">
    import type { OpenDocument } from './document';

    interface Props {
        tabs: OpenDocument[];
        activeId: string;
        // The Find in Files results view is a tab too, but it has no document
        // behind it, so it is rendered from these props instead of the list.
        searchOpen?: boolean;
        searchActive?: boolean;
        onSelect: (id: string) => void;
        onClose: (id: string) => void;
        onSelectSearch?: () => void;
        onCloseSearch?: () => void;
    }

    let {
        tabs,
        activeId,
        searchOpen = false,
        searchActive = false,
        onSelect,
        onClose,
        onSelectSearch,
        onCloseSearch,
    }: Props = $props();

    function label(tab: OpenDocument): string {
        return tab.path !== '' ? tab.path : `${tab.name} (unsaved)`;
    }
</script>

{#if tabs.length > 0 || searchOpen}
    <div class="tabs" role="tablist" aria-label="Open files">
        {#each tabs as tab (tab.id)}
            <div class="tab" class:tab--active={tab.id === activeId}>
                <button
                    class="tab__label"
                    role="tab"
                    aria-selected={tab.id === activeId}
                    title={label(tab)}
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
                    <svg aria-hidden="true" height="8" viewBox="0 0 8 8" width="8">
                        <path d="M1 1 L7 7 M7 1 L1 7" fill="none" stroke="currentColor" stroke-width="1.2" />
                    </svg>
                </button>
            </div>
        {/each}
        {#if searchOpen}
            <div class="tab" class:tab--active={searchActive}>
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
                    <svg aria-hidden="true" height="8" viewBox="0 0 8 8" width="8">
                        <path d="M1 1 L7 7 M7 1 L1 7" fill="none" stroke="currentColor" stroke-width="1.2" />
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
        width: 22px;
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
