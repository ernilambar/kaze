<script lang="ts">
    import { onMount } from 'svelte';

    interface Props {
        title: string;
        // label is what the field is asking for, e.g. the folder being created
        // in. It is printed above the input when non-empty; a title that already
        // names the entry leaves it empty.
        label?: string;
        // value is the initial text, selected on open so a rename or a copy is
        // one keystroke away from being accepted.
        value: string;
        // selectStem selects only the part before the extension, so typing
        // replaces the name and keeps the file type, the way Finder does.
        selectStem?: boolean;
        // selectFolder selects everything before the last path separator, so
        // typing replaces the folder and keeps the file name.
        selectFolder?: boolean;
        // error is the backend's refusal, shown in place so the dialog can be
        // corrected instead of reopened.
        error?: string;
        onSubmit: (value: string) => void;
        onCancel: () => void;
    }

    let {
        title,
        label = '',
        value = $bindable(''),
        selectStem = false,
        selectFolder = false,
        error = '',
        onSubmit,
        onCancel,
    }: Props = $props();

    let input = $state<HTMLInputElement>();

    // The field takes the text and selects what the user most likely wants to
    // replace: the whole name, or the stem of a file being renamed. A full path
    // keeps its folders and selects only the last segment's stem.
    onMount(() => {
        input?.focus();
        const text = input?.value ?? '';
        if (selectStem && text !== '') {
            const base = Math.max(text.lastIndexOf('/'), text.lastIndexOf('\\')) + 1;
            const dot = text.lastIndexOf('.');
            input?.setSelectionRange(base, dot > base ? dot : text.length);
            return;
        }
        if (selectFolder) {
            const slash = Math.max(text.lastIndexOf('/'), text.lastIndexOf('\\'));
            if (slash > 0) {
                input?.setSelectionRange(0, slash);
                return;
            }
        }
        input?.select();
    });

    function submit(event: SubmitEvent): void {
        event.preventDefault();
        const next = value.trim();
        if (next === '') {
            return;
        }
        onSubmit(next);
    }

    function onKeydown(event: KeyboardEvent): void {
        if (event.key === 'Escape') {
            event.preventDefault();
            onCancel();
        }
    }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="prompt">
    <button
        class="prompt__backdrop"
        aria-label={`Cancel ${title}`}
        tabindex="-1"
        onclick={onCancel}
    ></button>
    <div
        class="prompt__dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="name-title"
        tabindex="-1"
    >
        <h2 class="prompt__title" id="name-title">{title}</h2>
        <form onsubmit={submit}>
            {#if label !== ''}
                <label class="prompt__label" for="name-input">{label}</label>
            {/if}
            <input
                class="prompt__input"
                id="name-input"
                bind:this={input}
                bind:value
                spellcheck="false"
                autocomplete="off"
            />
            {#if error !== ''}
                <p class="prompt__error">{error}</p>
            {/if}
            <div class="prompt__actions">
                <button class="prompt__button" type="button" onclick={onCancel}>Cancel</button>
                <button
                    class="prompt__button prompt__button--primary"
                    type="submit"
                    disabled={value.trim() === ''}
                >{title}</button>
            </div>
        </form>
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
        margin: 0 0 12px;
        font-size: var(--ui-font-size);
        font-weight: 600;
    }

    .prompt__label {
        display: block;
        margin-bottom: 6px;
        color: var(--fg-muted);
        font-size: calc(var(--ui-font-size) - 1px);
    }

    .prompt__input {
        width: 100%;
        padding: 5px 8px;
        border: 1px solid var(--border);
        border-radius: 4px;
        background: var(--bg);
        color: var(--fg);
        font: inherit;
    }

    .prompt__input:focus-visible {
        outline: 1px solid var(--fg-muted);
        outline-offset: 0;
    }

    .prompt__error {
        margin: 8px 0 0;
        color: var(--fg-danger);
        font-size: calc(var(--ui-font-size) - 1px);
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

    .prompt__button:hover:enabled {
        background: var(--bg-hover);
    }

    .prompt__button--primary {
        border-color: var(--fg-muted);
        color: var(--bg);
        background: var(--fg);
    }

    .prompt__button--primary:hover:enabled {
        filter: brightness(1.15);
    }

    .prompt__button:disabled {
        cursor: default;
        opacity: 0.5;
    }

    .prompt__button:focus-visible {
        outline: 1px solid var(--fg);
        outline-offset: 1px;
    }
</style>
