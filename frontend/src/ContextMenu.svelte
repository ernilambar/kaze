<script lang="ts">
    import { onMount } from 'svelte';
    import type { ContextMenuItem } from './contextmenu';

    interface Props {
        x: number;
        y: number;
        items: ContextMenuItem[];
        onSelect: (id: string) => void;
        onClose: () => void;
    }

    let { x, y, items, onSelect, onClose }: Props = $props();

    let menu = $state<HTMLDivElement>();
    // The menu opens at the pointer, but a click near the window edge would put
    // part of it off screen, so it is pulled back inside after being measured.
    // svelte-ignore state_referenced_locally
    let left = $state(x);
    // svelte-ignore state_referenced_locally
    let top = $state(y);

    onMount(() => {
        const rect = menu?.getBoundingClientRect();
        if (rect === undefined) {
            return;
        }
        left = Math.min(x, window.innerWidth - rect.width - 4);
        top = Math.min(y, window.innerHeight - rect.height - 4);
    });

    function onPointerDown(event: PointerEvent): void {
        if (menu !== undefined && !menu.contains(event.target as Node)) {
            onClose();
        }
    }

    function onKeydown(event: KeyboardEvent): void {
        if (event.key === 'Escape') {
            event.preventDefault();
            onClose();
        }
    }
</script>

<svelte:window onpointerdown={onPointerDown} onkeydown={onKeydown} />

<div class="menu" role="menu" bind:this={menu} style:left={`${left}px`} style:top={`${top}px`}>
    {#each items as item, index (item.id === '' ? `separator-${index}` : item.id)}
        {#if item.id === ''}
            <div class="menu__separator" role="separator"></div>
        {:else}
            <button
                class="menu__item"
                role="menuitem"
                type="button"
                disabled={item.disabled === true}
                onclick={() => onSelect(item.id)}
            ><span>{item.label}</span>{#if item.shortcut !== undefined}<span class="menu__shortcut">{item.shortcut}</span>{/if}</button>
        {/if}
    {/each}
</div>

<style>
    .menu {
        position: fixed;
        z-index: 200;
        min-width: 180px;
        padding: 4px;
        border: 1px solid var(--border);
        border-radius: 6px;
        background: var(--bg-raised);
        box-shadow: 0 8px 24px rgba(0, 0, 0, 0.45);
    }

    .menu__item {
        display: flex;
        justify-content: space-between;
        gap: 24px;
        width: 100%;
        padding: 4px 10px;
        border: none;
        border-radius: 4px;
        background: none;
        color: var(--fg);
        font: inherit;
        text-align: left;
        white-space: nowrap;
        cursor: pointer;
    }

    .menu__item:hover:enabled,
    .menu__item:focus-visible:enabled {
        background: var(--bg-hover);
        outline: none;
    }

    .menu__item:disabled {
        color: var(--fg-muted);
        opacity: 0.6;
        cursor: default;
    }

    .menu__shortcut {
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
    }

    .menu__separator {
        height: 1px;
        margin: 4px 6px;
        background: var(--border);
    }
</style>
