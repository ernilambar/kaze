<script lang="ts">
    import { onMount } from 'svelte';
    import type { ThemeInfo } from './theme';
    import type {
        ThemeMode,
        ThemeSettings,
        UISettings,
    } from './settings';

    interface Props {
        theme: ThemeSettings;
        ui: UISettings;
        themes: ThemeInfo[];
        warning?: string;
        onTheme: (next: ThemeSettings) => void;
        onUI: (next: UISettings) => void;
        onOpenSettings: () => void;
        onOpenDefaults: () => void;
        onOpenThemes: () => void;
        onOpenKeybindings: () => void;
        onClose: () => void;
    }

    let {
        theme,
        ui,
        themes,
        warning = '',
        onTheme,
        onUI,
        onOpenSettings,
        onOpenDefaults,
        onOpenThemes,
        onOpenKeybindings,
        onClose,
    }: Props = $props();

    let dialog = $state<HTMLElement>();

    let lightThemes = $derived(themes.filter((entry) => entry.appearance === 'light'));
    let darkThemes = $derived(themes.filter((entry) => entry.appearance === 'dark'));

    function setMode(mode: string): void {
        if (mode === 'system' || mode === 'light' || mode === 'dark') {
            onTheme({ ...theme, mode: mode as ThemeMode });
        }
    }

    function setLight(name: string): void {
        onTheme({ ...theme, light: name });
    }

    function setDark(name: string): void {
        onTheme({ ...theme, dark: name });
    }

    function setFontFamily(value: string): void {
        onUI({ ...ui, fontFamily: value });
    }

    function setFontSize(value: string): void {
        const parsed = Number(value);
        if (Number.isFinite(parsed) && parsed > 0) {
            onUI({ ...ui, fontSize: parsed });
        }
    }

    function setTabIcons(checked: boolean): void {
        onUI({ ...ui, tabIcons: checked });
    }

    function setIconStyle(value: string): void {
        onUI({ ...ui, iconStyle: value === 'monochrome' ? 'monochrome' : 'colorful' });
    }

    function setSidebarPosition(value: string): void {
        onUI({ ...ui, sidebarPosition: value === 'right' ? 'right' : 'left' });
    }

    function onKeydown(event: KeyboardEvent): void {
        if (event.key === 'Escape') {
            event.preventDefault();
            onClose();
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
            <button class="settings__close" aria-label="Close Settings" onclick={onClose}>
                <svg aria-hidden="true" height="9" viewBox="0 0 9 9" width="9">
                    <path d="M1 1 L8 8 M8 1 L1 8" fill="none" stroke="currentColor" stroke-width="1.2" />
                </svg>
            </button>
        </header>

        <div class="settings__body">
            <section class="settings__section">
                <h3>Appearance</h3>
                <label class="settings__row">
                    <span>Mode</span>
                    <select value={theme.mode} onchange={(event) => setMode(event.currentTarget.value)}>
                        <option value="system">System</option>
                        <option value="light">Light</option>
                        <option value="dark">Dark</option>
                    </select>
                </label>
                <label class="settings__row">
                    <span>Light theme</span>
                    <select value={theme.light} onchange={(event) => setLight(event.currentTarget.value)}>
                        {#each lightThemes as entry (entry.name)}
                            <option value={entry.name}>{entry.family} — {entry.name}</option>
                        {/each}
                    </select>
                </label>
                <label class="settings__row">
                    <span>Dark theme</span>
                    <select value={theme.dark} onchange={(event) => setDark(event.currentTarget.value)}>
                        {#each darkThemes as entry (entry.name)}
                            <option value={entry.name}>{entry.family} — {entry.name}</option>
                        {/each}
                    </select>
                </label>
            </section>

            <section class="settings__section">
                <h3>Interface</h3>
                <label class="settings__row">
                    <span>UI font</span>
                    <input
                        type="text"
                        value={ui.fontFamily}
                        placeholder="System font"
                        spellcheck="false"
                        onchange={(event) => setFontFamily(event.currentTarget.value)}
                    />
                </label>
                <label class="settings__row">
                    <span>UI size</span>
                    <input
                        type="number"
                        min="9"
                        max="48"
                        step="1"
                        value={ui.fontSize}
                        onchange={(event) => setFontSize(event.currentTarget.value)}
                    />
                </label>
                <label class="settings__row">
                    <span>Tab icons</span>
                    <input
                        type="checkbox"
                        checked={ui.tabIcons}
                        onchange={(event) => setTabIcons(event.currentTarget.checked)}
                    />
                </label>
                <label class="settings__row">
                    <span>Icon style</span>
                    <select value={ui.iconStyle} onchange={(event) => setIconStyle(event.currentTarget.value)}>
                        <option value="colorful">Colorful</option>
                        <option value="monochrome">Monochrome</option>
                    </select>
                </label>
                <label class="settings__row">
                    <span>Sidebar</span>
                    <select
                        value={ui.sidebarPosition}
                        onchange={(event) => setSidebarPosition(event.currentTarget.value)}
                    >
                        <option value="left">Left</option>
                        <option value="right">Right</option>
                    </select>
                </label>
            </section>

            <section class="settings__section">
                <h3>Files</h3>
                <p class="settings__hint">
                    Editor options, excludes and workspace state live in settings.json.
                </p>
                <div class="settings__buttons">
                    <button type="button" onclick={onOpenSettings}>Settings JSON</button>
                    <button type="button" onclick={onOpenDefaults}>Default Settings</button>
                    <button type="button" onclick={onOpenThemes}>Themes Folder</button>
                    <button type="button" onclick={onOpenKeybindings}>Keybindings</button>
                </div>
            </section>

            {#if warning !== ''}
                <p class="settings__warning">{warning}</p>
            {/if}
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
        background: rgba(0, 0, 0, 0.35);
        cursor: default;
    }

    .settings__dialog {
        position: relative;
        display: flex;
        flex-direction: column;
        width: min(560px, calc(100vw - 48px));
        max-height: min(640px, calc(100vh - 64px));
        border: 1px solid var(--border);
        border-radius: 6px;
        background: var(--bg-raised);
        box-shadow: 0 12px 32px rgba(0, 0, 0, 0.45);
        outline: none;
    }

    .settings__head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex: 0 0 auto;
        padding: 12px 16px;
        border-bottom: 1px solid var(--border);
    }

    .settings__head h2 {
        margin: 0;
        font-size: var(--ui-font-size);
        font-weight: 600;
    }

    .settings__close {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 22px;
        height: 22px;
        padding: 0;
        border: none;
        border-radius: 3px;
        background: none;
        color: var(--fg-muted);
        cursor: pointer;
    }

    .settings__close:hover {
        background: var(--bg-hover);
        color: var(--fg);
    }

    .settings__body {
        padding: 4px 16px 16px;
        overflow-y: auto;
    }

    .settings__section {
        padding-top: 12px;
    }

    .settings__section h3 {
        margin: 0 0 8px;
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
        font-weight: 600;
        letter-spacing: 0.06em;
        text-transform: uppercase;
    }

    .settings__row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
        min-height: 28px;
    }

    .settings__row select,
    .settings__row input[type='text'],
    .settings__row input[type='number'] {
        width: 260px;
        min-width: 0;
        padding: 3px 6px;
        border: 1px solid var(--border);
        border-radius: 3px;
        background: var(--bg);
        color: var(--fg);
        font: inherit;
        outline: none;
    }

    .settings__row select:focus,
    .settings__row input:focus {
        border-color: var(--fg-muted);
    }

    .settings__hint {
        margin: 0 0 8px;
        color: var(--fg-muted);
    }

    .settings__buttons {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
    }

    .settings__buttons button {
        padding: 4px 10px;
        border: 1px solid var(--border);
        border-radius: 4px;
        background: var(--bg);
        color: var(--fg);
        font: inherit;
        cursor: pointer;
    }

    .settings__buttons button:hover {
        background: var(--bg-hover);
    }

    .settings__warning {
        margin: 12px 0 0;
        color: var(--fg-danger);
    }
</style>
