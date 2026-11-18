<script lang="ts">
    import type { Snippet } from 'svelte';
    import ResetButton from './ResetButton.svelte';

    // group marks a control made of several inputs (a radio group). The row is
    // then a plain container: a wrapping label would hand every click to the
    // first input.
    interface Props {
        name: string;
        description: string;
        control: Snippet;
        modified?: boolean;
        onReset?: () => void;
        group?: boolean;
    }

    let { name, description, control, modified = false, onReset, group = false }: Props =
        $props();
</script>

{#snippet text()}
    <span class="settings__text">
        <span class="settings__name">{name}</span>
        <span class="settings__description">{description}</span>
    </span>
{/snippet}

<div class="settings__row" class:settings__row--static={group}>
    {#if group}
        <div class="settings__row-main">
            {@render text()}
            {@render control()}
        </div>
    {:else}
        <label class="settings__row-main">
            {@render text()}
            <span class="settings__control">{@render control()}</span>
        </label>
    {/if}
    {#if modified && onReset !== undefined}
        <ResetButton label={name} onclick={onReset} />
    {/if}
</div>

<style>
    .settings__row {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 7px 8px;
        margin: 0 -8px;
        border-radius: 6px;
    }

    /* Each row is its own instance, so the previous row is outside this
       component's scope as far as the compiler can tell. */
    :global(.settings__row) + .settings__row {
        box-shadow: inset 0 1px 0 var(--border);
    }

    .settings__row:hover {
        background: var(--bg-hover);
    }

    .settings__row-main {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 18px;
        flex: 1 1 auto;
        min-width: 0;
        cursor: pointer;
    }

    .settings__row--static .settings__row-main {
        cursor: default;
    }

    .settings__text {
        display: flex;
        flex-direction: column;
        gap: 2px;
        min-width: 0;
    }

    .settings__name {
        color: var(--fg);
    }

    .settings__description {
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
        line-height: 1.35;
    }

    /* Controls come from the sections' snippets, so they carry the section's
       scope; every control style lives here, global under the control slot. */
    .settings__control {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        flex: 0 0 auto;
    }

    .settings__control :global(input[type='text']),
    .settings__control :global(input[type='number']),
    .settings__control :global(select) {
        box-sizing: border-box;
        border: 1px solid var(--border);
        border-radius: 5px;
        background: var(--bg);
        color: var(--fg);
        font: inherit;
        outline: none;
        transition:
            border-color 120ms ease,
            background 120ms ease;
    }

    .settings__control :global(input[type='text']) {
        width: 260px;
        padding: 5px 8px;
    }

    .settings__control :global(input.settings__font-input) {
        font-family: var(--editor-font-family, monospace);
    }

    .settings__control :global(input[type='number']) {
        width: 76px;
        padding: 5px 6px 5px 8px;
    }

    .settings__control :global(select) {
        width: 190px;
        padding: 5px 28px 5px 8px;
        appearance: none;
        -webkit-appearance: none;
    }

    .settings__control :global(input:hover),
    .settings__control :global(select:hover) {
        border-color: var(--fg-muted);
    }

    .settings__control :global(input:focus),
    .settings__control :global(select:focus) {
        border-color: var(--link, #4daafc);
    }

    .settings__control :global(.settings__field) {
        display: inline-flex;
        align-items: center;
        gap: 6px;
    }

    .settings__control :global(.settings__unit) {
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
    }

    .settings__control :global(.settings__switch) {
        appearance: none;
        -webkit-appearance: none;
        position: relative;
        flex: 0 0 auto;
        width: 32px;
        height: 18px;
        margin: 0;
        border-radius: 9px;
        background: var(--border);
        cursor: pointer;
        transition: background 120ms ease;
    }

    .settings__control :global(.settings__switch::after) {
        content: '';
        position: absolute;
        top: 2px;
        left: 2px;
        width: 14px;
        height: 14px;
        border-radius: 50%;
        background: #ffffff;
        box-shadow: 0 1px 2px rgba(0, 0, 0, 0.4);
        transition: transform 120ms ease;
    }

    .settings__control :global(.settings__switch:checked) {
        background: var(--link, #4daafc);
    }

    .settings__control :global(.settings__switch:checked::after) {
        transform: translateX(14px);
    }

    .settings__control :global(.settings__switch:focus-visible) {
        outline: 2px solid var(--link, #4daafc);
        outline-offset: 2px;
    }

    .settings__control :global(.settings__select) {
        position: relative;
        display: inline-flex;
        align-items: center;
    }

    .settings__control :global(.settings__select-arrow) {
        position: absolute;
        right: 8px;
        top: 50%;
        transform: translateY(-50%);
        color: var(--fg-muted);
        pointer-events: none;
    }
</style>
