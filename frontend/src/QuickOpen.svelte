<script lang="ts">
    import { onMount } from 'svelte';
    import { rankQuickOpen, type QuickOpenFile } from './quickOpen';

    interface Props {
        files: QuickOpenFile[];
        loading?: boolean;
        onOpen: (path: string) => void;
        onClose: () => void;
    }

    let { files, loading = false, onOpen, onClose }: Props = $props();

    let query = $state('');
    let selected = $state(0);
    let input = $state<HTMLInputElement>();
    let list = $state<HTMLUListElement>();

    let results = $derived(rankQuickOpen(query, files));

    function move(step: number): void {
        if (results.length === 0) {
            return;
        }
        selected = Math.min(Math.max(selected + step, 0), results.length - 1);
    }

    function choose(): void {
        const file = results[selected] ?? results[0];
        if (file !== undefined) {
            onOpen(file.path);
        }
    }

    function onKeydown(event: KeyboardEvent): void {
        switch (event.key) {
            case 'ArrowDown':
                event.preventDefault();
                move(1);
                break;
            case 'ArrowUp':
                event.preventDefault();
                move(-1);
                break;
            case 'Enter':
                event.preventDefault();
                choose();
                break;
            case 'Escape':
                event.preventDefault();
                onClose();
                break;
        }
    }

    // Keep the selected row visible when the arrow keys move past the list's
    // scroll viewport.
    $effect(() => {
        selected;
        list?.querySelector('[aria-selected="true"]')?.scrollIntoView({ block: 'nearest' });
    });

    onMount(() => input?.focus());
</script>

<div class="quick-open">
    <button
        class="quick-open__backdrop"
        aria-label="Close Quick Open"
        tabindex="-1"
        onclick={onClose}
    ></button>
    <div class="quick-open__dialog" role="dialog" aria-modal="true" aria-label="Quick Open">
        <input
            class="quick-open__input"
            bind:this={input}
            bind:value={query}
            oninput={() => (selected = 0)}
            onkeydown={onKeydown}
            type="text"
            placeholder="Search files by name or path"
            aria-label="Search files by name or path"
            aria-controls="quick-open-results"
            aria-activedescendant={results.length > 0 ? `quick-open-option-${selected}` : undefined}
            autocomplete="off"
            spellcheck="false"
        />
        <ul class="quick-open__results" id="quick-open-results" role="listbox" bind:this={list}>
            {#each results as file, index (file.path)}
                <li role="presentation">
                    <button
                        id={`quick-open-option-${index}`}
                        class="quick-open__option"
                        type="button"
                        role="option"
                        aria-selected={index === selected}
                        onclick={() => onOpen(file.path)}
                        onmouseenter={() => (selected = index)}
                        title={file.path}
                    >
                        <span class="quick-open__name">{file.name}</span>
                        {#if file.detail !== ''}
                            <span class="quick-open__detail">{file.detail}</span>
                        {/if}
                    </button>
                </li>
            {/each}
        </ul>
        {#if results.length === 0}
            <p class="quick-open__empty">
                {#if loading}
                    Loading files…
                {:else if files.length === 0}
                    Add a folder to the sidebar to search its files.
                {:else}
                    No matching files.
                {/if}
            </p>
        {/if}
    </div>
</div>

<style>
    .quick-open {
        position: fixed;
        inset: 0;
        z-index: 100;
        display: flex;
        justify-content: center;
        align-items: flex-start;
    }

    .quick-open__backdrop {
        position: absolute;
        inset: 0;
        padding: 0;
        border: none;
        background: rgba(0, 0, 0, 0.35);
        cursor: default;
    }

    .quick-open__dialog {
        position: relative;
        width: min(640px, 90vw);
        margin-top: 10vh;
        background: var(--bg-raised);
        border: 1px solid var(--border);
        border-radius: 6px;
        box-shadow: 0 12px 32px rgba(0, 0, 0, 0.45);
        overflow: hidden;
    }

    .quick-open__input {
        width: 100%;
        padding: 10px 12px;
        border: none;
        border-bottom: 1px solid var(--border);
        background: var(--bg-sidebar);
        color: var(--fg);
        font: inherit;
        font-size: 14px;
        outline: none;
    }

    .quick-open__input::placeholder {
        color: var(--fg-muted);
    }

    .quick-open__results {
        max-height: 46vh;
        margin: 0;
        padding: 4px 0;
        list-style: none;
        overflow-y: auto;
    }

    .quick-open__option {
        display: flex;
        align-items: baseline;
        gap: 8px;
        width: 100%;
        padding: 5px 12px;
        border: none;
        background: none;
        color: var(--fg);
        font: inherit;
        text-align: left;
        cursor: pointer;
    }

    .quick-open__option[aria-selected='true'] {
        background: var(--bg-selected);
    }

    .quick-open__name {
        flex: 0 1 auto;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .quick-open__detail {
        flex: 0 1 auto;
        margin-left: auto;
        color: var(--fg-muted);
        font-size: 11px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .quick-open__empty {
        margin: 0;
        padding: 12px;
        color: var(--fg-muted);
    }
</style>
