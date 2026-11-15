<script lang="ts">
    import { onMount } from 'svelte';

    interface Action {
        id: string;
        label: string;
        // primary is the action Return triggers; the dialog focuses it.
        primary?: boolean;
    }

    interface Props {
        title: string;
        message: string;
        // details names the buffers or paths the question is about.
        details?: string[];
        actions: Action[];
        onSelect: (id: string) => void;
        onCancel: () => void;
    }

    let { title, message, details = [], actions, onSelect, onCancel }: Props = $props();

    let dialog = $state<HTMLDivElement>();

    // Escape closes the dialog, the same as clicking its backdrop. Return is
    // left to the focused button, so it cannot fire an action twice.
    function onKeydown(event: KeyboardEvent): void {
        if (event.key === 'Escape') {
            event.preventDefault();
            onCancel();
        }
    }

    onMount(() => {
        const primary = dialog?.querySelector<HTMLButtonElement>('[data-primary="true"]');
        (primary ?? dialog)?.focus();
    });
</script>

<svelte:window onkeydown={onKeydown} />

<div class="prompt">
    <button class="prompt__backdrop" aria-label={`Cancel ${title}`} tabindex="-1" onclick={onCancel}></button>
    <div
        class="prompt__dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="prompt-title"
        aria-describedby="prompt-message"
        tabindex="-1"
        bind:this={dialog}
    >
        <h2 class="prompt__title" id="prompt-title">{title}</h2>
        <p class="prompt__message" id="prompt-message">{message}</p>
        {#if details.length > 0}
            <ul class="prompt__details">
                {#each details as detail (detail)}
                    <li>{detail}</li>
                {/each}
            </ul>
        {/if}
        <div class="prompt__actions">
            {#each actions as action (action.id)}
                <button
                    class="prompt__button"
                    class:prompt__button--primary={action.primary === true}
                    type="button"
                    data-primary={action.primary === true}
                    onclick={() => onSelect(action.id)}
                >{action.label}</button>
            {/each}
        </div>
    </div>
</div>

<style>
    .prompt {
        position: fixed;
        inset: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 100;
    }

    .prompt__backdrop {
        position: absolute;
        inset: 0;
        border: none;
        background: rgba(0, 0, 0, 0.35);
        cursor: default;
    }

    .prompt__dialog {
        position: relative;
        width: min(420px, calc(100vw - 48px));
        padding: 16px;
        border: 1px solid var(--border);
        border-radius: 6px;
        background: var(--bg-raised);
        box-shadow: 0 12px 32px rgba(0, 0, 0, 0.45);
    }

    .prompt__dialog:focus-visible {
        outline: none;
    }

    .prompt__title {
        margin: 0 0 8px;
        font-size: 13px;
        font-weight: 600;
    }

    .prompt__message {
        margin: 0;
        color: var(--fg-muted);
        line-height: 1.4;
    }

    .prompt__details {
        margin: 8px 0 0;
        padding-left: 18px;
        color: var(--fg-muted);
        max-height: 160px;
        overflow-y: auto;
    }

    .prompt__actions {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
        margin-top: 16px;
    }

    .prompt__button {
        padding: 4px 12px;
        border: 1px solid var(--border);
        border-radius: 4px;
        background: var(--bg);
        color: var(--fg);
        font: inherit;
        cursor: pointer;
    }

    .prompt__button:hover {
        background: var(--bg-hover);
    }

    .prompt__button--primary {
        border-color: var(--fg-muted);
        color: var(--bg);
        background: var(--fg);
    }

    .prompt__button--primary:hover {
        filter: brightness(1.15);
    }

    .prompt__button:focus-visible {
        outline: 1px solid var(--fg);
        outline-offset: 1px;
    }
</style>
