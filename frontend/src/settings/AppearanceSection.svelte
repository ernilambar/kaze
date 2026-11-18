<script lang="ts">
    import type { ThemeInfo } from '../theme';
    import { defaultThemeSettings, type ThemeMode, type ThemeSettings } from '../settings';
    import ResetButton from './ResetButton.svelte';
    import SettingRow from './SettingRow.svelte';

    interface Props {
        theme: ThemeSettings;
        themes: ThemeInfo[];
        onTheme: (next: ThemeSettings) => void;
    }

    let { theme, themes, onTheme }: Props = $props();

    const THEME_MODES: readonly { id: ThemeMode; label: string }[] = [
        { id: 'system', label: 'System' },
        { id: 'light', label: 'Light' },
        { id: 'dark', label: 'Dark' },
    ];

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
</script>

{#snippet themeGroup(
    title: string,
    appearance: 'light' | 'dark',
    list: ThemeInfo[],
    selected: string,
    fallback: string,
    apply: (name: string) => void,
)}
    <div class="settings__theme-group">
        <div class="settings__theme-head">
            <h4>{title}</h4>
            {#if selected !== fallback}
                <ResetButton label={title} onclick={() => apply(fallback)} />
            {/if}
        </div>
        {#if list.length === 0}
            <p class="settings__hint">No {appearance} themes are installed.</p>
        {:else}
            <div class="settings__theme-grid">
                {#each list as entry (entry.name)}
                    <button
                        type="button"
                        class="settings__theme-card"
                        class:settings__theme-card--active={entry.name === selected}
                        aria-pressed={entry.name === selected}
                        aria-label={`Use ${entry.name} as the ${title.toLowerCase()}`}
                        onclick={() => apply(entry.name)}
                    >
                        <span
                            class="settings__theme-preview"
                            style:background-color={entry.preview.background || 'var(--bg)'}
                            style:color={entry.preview.foreground || 'var(--fg)'}
                        >
                            <span class="settings__theme-code">
                                <span style:color={entry.preview.comment || 'inherit'}>// Kaze</span>
                            </span>
                            <span class="settings__theme-code">
                                <span style:color={entry.preview.keyword || 'inherit'}>const</span>
                                name
                                <span style:color={entry.preview.foreground || 'inherit'}>=</span>
                                <span style:color={entry.preview.string || 'inherit'}>"kaze"</span>;
                            </span>
                            <span class="settings__theme-code">
                                <span style:color={entry.preview.accent || 'inherit'}>/**/</span>
                            </span>
                        </span>
                        <span class="settings__theme-meta">
                            <span class="settings__theme-name">{entry.name}</span>
                            <span class="settings__theme-family">{entry.family}</span>
                        </span>
                        {#if entry.name === selected}
                            <span class="settings__theme-check" aria-hidden="true">
                                <svg height="10" viewBox="0 0 10 10" width="10">
                                    <path
                                        d="M1.5 5.3 4 7.7 8.5 2.5"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-linecap="round"
                                        stroke-linejoin="round"
                                        stroke-width="1.5"
                                    />
                                </svg>
                            </span>
                        {/if}
                    </button>
                {/each}
            </div>
        {/if}
    </div>
{/snippet}

<section class="settings__section">
    <div class="settings__section-head">
        <h3>Appearance</h3>
        {#if theme.mode !== defaultThemeSettings.mode ||
            theme.light !== defaultThemeSettings.light ||
            theme.dark !== defaultThemeSettings.dark}
            <ResetButton label="Appearance" onclick={() => onTheme({ ...defaultThemeSettings })} />
        {/if}
    </div>
    <p class="settings__hint">
        The light and dark palettes are used by the matching appearance.
    </p>
    <SettingRow name="Mode" description="System follows the macOS appearance." group>
        {#snippet control()}
            <div class="settings__segmented" role="group" aria-label="Appearance mode">
                {#each THEME_MODES as mode (mode.id)}
                    <label
                        class="settings__segment"
                        class:settings__segment--active={theme.mode === mode.id}
                    >
                        <input
                            type="radio"
                            name="theme-mode"
                            value={mode.id}
                            checked={theme.mode === mode.id}
                            onchange={() => setMode(mode.id)}
                        />
                        {mode.label}
                    </label>
                {/each}
            </div>
        {/snippet}
    </SettingRow>
    <div class="settings__themes">
        {@render themeGroup(
            'Light theme',
            'light',
            lightThemes,
            theme.light,
            defaultThemeSettings.light,
            setLight,
        )}
        {@render themeGroup(
            'Dark theme',
            'dark',
            darkThemes,
            theme.dark,
            defaultThemeSettings.dark,
            setDark,
        )}
    </div>
</section>

<style>
    .settings__segmented {
        display: inline-flex;
        padding: 2px;
        border: 1px solid var(--border);
        border-radius: 6px;
        background: var(--bg);
    }

    .settings__segment {
        position: relative;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        min-width: 64px;
        padding: 4px 10px;
        border-radius: 4px;
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
        cursor: pointer;
        transition:
            background 120ms ease,
            color 120ms ease;
    }

    .settings__segment input {
        position: absolute;
        opacity: 0;
        pointer-events: none;
    }

    .settings__segment:hover {
        color: var(--fg);
    }

    .settings__segment--active {
        background: var(--bg-selected);
        color: var(--fg);
        font-weight: 600;
    }

    .settings__segment:focus-within {
        outline: 2px solid var(--link, #4daafc);
        outline-offset: 1px;
    }

    .settings__themes {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(min(240px, 100%), 1fr));
        gap: 18px;
        align-items: start;
        margin-top: 14px;
    }

    .settings__theme-group {
        min-width: 0;
    }

    .settings__theme-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        margin-bottom: 8px;
    }

    .settings__theme-head h4 {
        margin: 0;
        color: var(--fg);
        font-size: var(--ui-font-size);
        font-weight: 600;
    }

    .settings__theme-grid {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .settings__theme-card {
        position: relative;
        display: flex;
        flex-direction: column;
        min-width: 0;
        padding: 0;
        border: 1px solid var(--border);
        border-radius: 7px;
        background: var(--bg);
        color: var(--fg);
        text-align: left;
        cursor: pointer;
        overflow: hidden;
        transition:
            border-color 120ms ease,
            background 120ms ease;
    }

    .settings__theme-card:hover {
        border-color: var(--fg-muted);
        background: var(--bg-hover);
    }

    .settings__theme-card--active {
        border-color: var(--link, #4daafc);
        box-shadow: 0 0 0 1px var(--link, #4daafc);
    }

    .settings__theme-card:focus-visible {
        outline: 2px solid var(--link, #4daafc);
        outline-offset: 2px;
    }

    .settings__theme-preview {
        display: flex;
        flex-direction: column;
        gap: 1px;
        height: 58px;
        padding: 8px 10px;
        overflow: hidden;
        font-family: var(--editor-font-family, monospace);
        font-size: 9px;
        line-height: 1.5;
    }

    .settings__theme-code {
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .settings__theme-meta {
        display: flex;
        flex-direction: column;
        gap: 1px;
        min-width: 0;
        padding: 7px 10px 9px;
    }

    .settings__theme-name {
        font-size: var(--ui-font-small);
        font-weight: 600;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .settings__theme-family {
        color: var(--fg-muted);
        font-size: var(--ui-font-tiny);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .settings__theme-check {
        position: absolute;
        top: 8px;
        right: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
        width: 18px;
        height: 18px;
        border-radius: 50%;
        background: var(--link, #4daafc);
        color: #ffffff;
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.4);
    }
</style>
