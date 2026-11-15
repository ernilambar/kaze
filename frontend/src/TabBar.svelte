<script lang="ts">
    import type { OpenDocument } from './document';

    interface Props {
        tabs: OpenDocument[];
        activePath: string;
        onSelect: (path: string) => void;
        onClose: (path: string) => void;
    }

    let { tabs, activePath, onSelect, onClose }: Props = $props();
</script>

{#if tabs.length > 0}
    <div class="tabs" role="tablist" aria-label="Open files">
        {#each tabs as tab (tab.path)}
            <div class="tab" class:tab--active={tab.path === activePath}>
                <button
                    class="tab__label"
                    role="tab"
                    aria-selected={tab.path === activePath}
                    title={tab.path}
                    onclick={() => onSelect(tab.path)}
                >{tab.name}</button>
                <button
                    class="tab__close"
                    aria-label={`Close ${tab.name}`}
                    title={`Close ${tab.name}`}
                    onclick={() => onClose(tab.path)}
                >
                    <svg aria-hidden="true" height="8" viewBox="0 0 8 8" width="8">
                        <path d="M1 1 L7 7 M7 1 L1 7" fill="none" stroke="currentColor" stroke-width="1.2" />
                    </svg>
                </button>
            </div>
        {/each}
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
    .tab--active .tab__close,
    .tab__close:focus-visible {
        opacity: 1;
    }

    .tab__close:hover {
        color: var(--fg);
    }
</style>
