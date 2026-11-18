<script lang="ts">
    import { onMount } from 'svelte';
    import type { ThemeInfo } from './theme';
    import type { EditorSettings, ThemeSettings, UISettings } from './settings';
    import AppearanceSection from './settings/AppearanceSection.svelte';
    import EditorSection from './settings/EditorSection.svelte';
    import FilesSection from './settings/FilesSection.svelte';
    import InterfaceSection from './settings/InterfaceSection.svelte';
    import SidebarSection from './settings/SidebarSection.svelte';

    type Section = 'editor' | 'appearance' | 'interface' | 'files' | 'sidebar';

    interface Props {
        editor: EditorSettings;
        theme: ThemeSettings;
        ui: UISettings;
        themes: ThemeInfo[];
        filesExclude: string[];
        trimTrailingWhitespace: boolean;
        searchExclude: string[];
        sidebarVisible: boolean;
        autoReveal: boolean;
        warning?: string;
        onEditor: (next: EditorSettings) => void;
        onTheme: (next: ThemeSettings) => void;
        onUI: (next: UISettings) => void;
        onFilesExclude: (next: string[]) => void;
        onTrimTrailingWhitespace: (trim: boolean) => void;
        onSearchExclude: (next: string[]) => void;
        onSidebarVisible: (visible: boolean) => void;
        onAutoReveal: (autoReveal: boolean) => void;
        onClose: () => void;
    }

    let {
        editor,
        theme,
        ui,
        themes,
        filesExclude,
        trimTrailingWhitespace,
        searchExclude,
        sidebarVisible,
        autoReveal,
        warning = '',
        onEditor,
        onTheme,
        onUI,
        onFilesExclude,
        onTrimTrailingWhitespace,
        onSearchExclude,
        onSidebarVisible,
        onAutoReveal,
        onClose,
    }: Props = $props();

    const SECTIONS: readonly { id: Section; label: string }[] = [
        { id: 'appearance', label: 'Appearance' },
        { id: 'editor', label: 'Editor' },
        { id: 'interface', label: 'Interface' },
        { id: 'files', label: 'Files' },
        { id: 'sidebar', label: 'Sidebar' },
    ];

    let dialog = $state<HTMLElement>();
    let section = $state<Section>('appearance');

    function focusableElements(): HTMLElement[] {
        if (dialog === undefined) {
            return [];
        }
        return Array.from(
            dialog.querySelectorAll<HTMLElement>(
                [
                    'button:not([disabled])',
                    'input:not([disabled]):not([type="radio"])',
                    'input[type="radio"]:checked',
                    'select:not([disabled])',
                    'textarea:not([disabled])',
                    '[tabindex]:not([tabindex="-1"])',
                ].join(','),
            ),
        ).filter((element) => element.getClientRects().length > 0);
    }

    // onKeydown closes on Escape and keeps Tab inside the dialog. A modal that
    // lets focus escape behind the backdrop is unusable with a keyboard.
    function onKeydown(event: KeyboardEvent): void {
        if (event.key === 'Escape') {
            event.preventDefault();
            onClose();
            return;
        }
        if (event.key !== 'Tab') {
            return;
        }
        const items = focusableElements();
        if (items.length === 0) {
            return;
        }
        const first = items[0];
        const last = items[items.length - 1];
        const active = document.activeElement;
        if (active === dialog) {
            event.preventDefault();
            (event.shiftKey ? last : first).focus();
            return;
        }
        if (!dialog?.contains(active)) {
            event.preventDefault();
            first.focus();
            return;
        }
        if (event.shiftKey && active === first) {
            event.preventDefault();
            last.focus();
        } else if (!event.shiftKey && active === last) {
            event.preventDefault();
            first.focus();
        }
    }

    onMount(() => dialog?.focus());
</script>

<svelte:window onkeydown={onKeydown} />

<div class="settings">
    <button
        class="settings__backdrop"
        aria-label="Close Settings"
        tabindex="-1"
        onclick={onClose}
    ></button>
    <div
        class="settings__dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="settings-title"
        tabindex="-1"
        bind:this={dialog}
    >
        <header class="settings__head">
            <h2 id="settings-title">Settings</h2>
            <div class="settings__head-actions">
                <span class="settings__autosave">Changes save automatically</span>
                <button class="settings__close" aria-label="Close Settings" onclick={onClose}>
                    <svg aria-hidden="true" height="9" viewBox="0 0 9 9" width="9">
                        <path
                            d="M1 1 L8 8 M8 1 L1 8"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="1.2"
                        />
                    </svg>
                </button>
            </div>
        </header>

        <div class="settings__layout">
            <nav class="settings__nav" aria-label="Settings sections">
                {#each SECTIONS as entry (entry.id)}
                    <button
                        type="button"
                        class="settings__nav-item"
                        class:settings__nav-item--active={section === entry.id}
                        aria-current={section === entry.id ? 'page' : undefined}
                        onclick={() => (section = entry.id)}
                    >
                        {entry.label}
                    </button>
                {/each}
            </nav>

            <div class="settings__body">
                {#if section === 'editor'}
                    <EditorSection {editor} {onEditor} />
                {:else if section === 'appearance'}
                    <AppearanceSection {theme} {themes} {onTheme} />
                {:else if section === 'interface'}
                    <InterfaceSection {ui} {onUI} />
                {:else if section === 'files'}
                    <FilesSection
                        {filesExclude}
                        {trimTrailingWhitespace}
                        {searchExclude}
                        {onFilesExclude}
                        {onTrimTrailingWhitespace}
                        {onSearchExclude}
                    />
                {:else if section === 'sidebar'}
                    <SidebarSection {sidebarVisible} {autoReveal} {onSidebarVisible} {onAutoReveal} />
                {/if}

                {#if warning !== ''}
                    <p class="settings__warning">{warning}</p>
                {/if}
            </div>
        </div>
    </div>
</div>

<style>
    .settings {
        position: fixed;
        inset: 0;
        z-index: 100;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .settings__backdrop {
        position: absolute;
        inset: 0;
        border: none;
        background: rgba(0, 0, 0, 0.4);
        cursor: default;
    }

    .settings__dialog {
        position: relative;
        display: flex;
        flex-direction: column;
        width: min(840px, calc(100vw - 48px));
        height: min(700px, calc(100vh - 64px));
        border: 1px solid var(--border);
        border-radius: 10px;
        background: var(--bg-raised);
        box-shadow: 0 24px 64px rgba(0, 0, 0, 0.5);
        outline: none;
        overflow: hidden;
    }

    .settings__head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
        flex: 0 0 auto;
        padding: 14px 18px;
        border-bottom: 1px solid var(--border);
    }

    .settings__head h2 {
        margin: 0;
        font-size: calc(var(--ui-font-size) + 1px);
        font-weight: 600;
        letter-spacing: -0.01em;
    }

    .settings__head-actions {
        display: flex;
        align-items: center;
        gap: 14px;
    }

    .settings__autosave {
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
    }

    .settings__close {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 24px;
        height: 24px;
        padding: 0;
        border: none;
        border-radius: 5px;
        background: none;
        color: var(--fg-muted);
        cursor: pointer;
    }

    .settings__close:hover {
        background: var(--bg-hover);
        color: var(--fg);
    }

    .settings__close:focus-visible {
        outline: 2px solid var(--link, #4daafc);
        outline-offset: 1px;
    }

    .settings__layout {
        display: flex;
        flex: 1 1 auto;
        min-height: 0;
    }

    .settings__nav {
        display: flex;
        flex-direction: column;
        gap: 2px;
        flex: 0 0 176px;
        padding: 12px 10px;
        border-right: 1px solid var(--border);
        background: var(--bg-sidebar);
        overflow-y: auto;
    }

    .settings__nav-item {
        width: 100%;
        padding: 7px 10px;
        border: none;
        border-radius: 5px;
        background: none;
        color: var(--fg-muted);
        font: inherit;
        text-align: left;
        cursor: pointer;
    }

    .settings__nav-item:hover {
        background: var(--bg-hover);
        color: var(--fg);
    }

    .settings__nav-item:focus-visible {
        outline: 2px solid var(--link, #4daafc);
        outline-offset: 1px;
    }

    .settings__nav-item--active {
        background: var(--bg-selected);
        box-shadow: inset 2px 0 0 var(--link, #4daafc);
        color: var(--fg);
        font-weight: 600;
    }

    .settings__body {
        flex: 1 1 auto;
        min-width: 0;
        padding: 2px 22px 24px;
        overflow-y: auto;
    }

    /* Sections render in child components; their shared layout is global
       under the body. Rows and controls style themselves in SettingRow. */
    .settings__body :global(.settings__section) {
        padding: 16px 0 6px;
    }

    .settings__body :global(.settings__section + .settings__section) {
        border-top: 1px solid var(--border);
    }

    .settings__body :global(.settings__section h3) {
        margin: 0 0 6px;
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
        font-weight: 600;
        letter-spacing: 0.06em;
        text-transform: uppercase;
    }

    .settings__body :global(.settings__section-head) {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
    }

    .settings__body :global(.settings__section-head h3) {
        margin: 0;
    }

    .settings__body :global(.settings__hint) {
        margin: 0 0 10px;
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
        line-height: 1.45;
    }

    .settings__warning {
        margin: 12px 0 0;
        color: var(--fg-danger);
    }
</style>
