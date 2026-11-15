<script lang="ts">
    import { onMount } from 'svelte';
    import type { PaletteItem } from './palette';

    interface Props {
        title: string;
        placeholder: string;
        rank: (query: string) => PaletteItem[];
        empty: string;
        onSelect: (id: string) => void;
        onClose: () => void;
    }

    let { title, placeholder, rank, empty, onSelect, onClose }: Props = $props();

    let query = $state('');
    let selected = $state(0);
    let input = $state<HTMLInputElement>();
    let list = $state<HTMLUListElement>();

    let results = $derived(rank(query));

    function move(step: number): void {
        if (results.length === 0) {
            return;
        }
        selected = Math.min(Math.max(selected + step, 0), results.length - 1);
    }

    function choose(): void {
        const item = results[selected] ?? results[0];
        if (item !== undefined) {
            onSelect(item.id);
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

<div class="palette">
    <button
        class="palette__backdrop"
        aria-label={`Close ${title}`}
        tabindex="-1"
        onclick={onClose}
    ></button>
    <div class="palette__dialog" role="dialog" aria-modal="true" aria-label={title}>
        <input
            class="palette__input"
            bind:this={input}
            bind:value={query}
            oninput={() => (selected = 0)}
            onkeydown={onKeydown}
            type="text"
            {placeholder}
            aria-label={placeholder}
            aria-controls="palette-results"
            aria-activedescendant={results.length > 0 ? `palette-option-${selected}` : undefined}
            autocomplete="off"
            spellcheck="false"
        />
        <ul class="palette__results" id="palette-results" role="listbox" bind:this={list}>
            {#each results as item, index (item.id)}
                <li role="presentation">
                    <button
                        id={`palette-option-${index}`}
                        class="palette__option"
                        type="button"
                        role="option"
                        aria-selected={index === selected}
                        onclick={() => onSelect(item.id)}
                        onmouseenter={() => (selected = index)}
                        title={item.detail === '' ? item.name : `${item.detail}/${item.name}`}
                    >
                        <span class="palette__name">{item.name}</span>
                        {#if item.detail !== ''}
                            <span class="palette__detail">{item.detail}</span>
                        {/if}
                    </button>
                </li>
            {/each}
        </ul>
        {#if results.length === 0}
            <p class="palette__empty">{empty}</p>
        {/if}
    </div>
</div>

<style>
    .palette {
        position: fixed;
        inset: 0;
        z-index: 100;
        display: flex;
        justify-content: center;
        align-items: flex-start;
    }

    .palette__backdrop {
        position: absolute;
        inset: 0;
        padding: 0;
        border: none;
        background: rgba(0, 0, 0, 0.35);
        cursor: default;
    }

    .palette__dialog {
        position: relative;
        width: min(640px, 90vw);
        margin-top: 10vh;
        background: var(--bg-raised);
        border: 1px solid var(--border);
        border-radius: 6px;
        box-shadow: 0 12px 32px rgba(0, 0, 0, 0.45);
        overflow: hidden;
    }

    .palette__input {
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

    .palette__input::placeholder {
        color: var(--fg-muted);
    }

    .palette__results {
        max-height: 46vh;
        margin: 0;
        padding: 4px 0;
        list-style: none;
        overflow-y: auto;
    }

    .palette__option {
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

    .palette__option[aria-selected='true'] {
        background: var(--bg-selected);
    }

    .palette__name {
        flex: 0 1 auto;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .palette__detail {
        flex: 0 1 auto;
        margin-left: auto;
        color: var(--fg-muted);
        font-size: 11px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .palette__empty {
        margin: 0;
        padding: 12px;
        color: var(--fg-muted);
    }
</style>
