<script lang="ts">
    import logo from './assets/kova.png';
    import type { PaletteItem } from './palette';

    interface Entry {
        command: string;
        label: string;
        keys: string;
    }

    interface Props {
        // recent holds the Open Recent entries, newest first. At most a few are
        // shown so the welcome screen stays one screenful.
        recent?: PaletteItem[];
        onCommand: (id: string) => void;
        onOpenRecent: (path: string) => void;
    }

    let { recent = [], onCommand, onOpenRecent }: Props = $props();

    // The key labels mirror the default accelerators in menu.go; a custom
    // keybinding.json entry may override one.
    const START: readonly Entry[] = [
        { command: 'new-file', label: 'New File', keys: '⌘N' },
        { command: 'open-file', label: 'Open File…', keys: '⌘O' },
        { command: 'add-folder', label: 'Add Folder to Sidebar…', keys: '⌘⇧O' },
        { command: 'open-recent', label: 'Open Recent…', keys: '⌃R' },
    ];

    const NAVIGATE: readonly Entry[] = [
        { command: 'quick-open', label: 'Quick Open…', keys: '⌘P' },
        { command: 'command-palette', label: 'Command Palette…', keys: '⌘⇧P' },
        { command: 'find-in-files', label: 'Find in Files…', keys: '⌘⇧F' },
        { command: 'toggle-sidebar', label: 'Toggle Sidebar', keys: '⌘B' },
        { command: 'settings', label: 'Settings…', keys: '⌘,' },
        { command: 'open-keybindings', label: 'Open Keybindings…', keys: '' },
    ];
</script>

<div class="welcome">
    <div class="welcome__content">
        <header class="welcome__brand">
            <img alt="" class="welcome__logo" src={logo} />
            <div>
                <h1 class="welcome__title">Kova</h1>
                <p class="welcome__tagline">A minimal text editor for macOS</p>
            </div>
        </header>
        <div class="welcome__grid">
            <section>
                <h2 class="welcome__heading">Start</h2>
                <ul class="welcome__list">
                    {#each START as entry (entry.command)}
                        <li>
                            <button class="welcome__item" onclick={() => onCommand(entry.command)}>
                                <span class="welcome__label">{entry.label}</span>
                                <kbd class="welcome__keys">{entry.keys}</kbd>
                            </button>
                        </li>
                    {/each}
                </ul>
            </section>
            <section>
                <h2 class="welcome__heading">Navigate</h2>
                <ul class="welcome__list">
                    {#each NAVIGATE as entry (entry.command)}
                        <li>
                            <button class="welcome__item" onclick={() => onCommand(entry.command)}>
                                <span class="welcome__label">{entry.label}</span>
                                {#if entry.keys !== ''}
                                    <kbd class="welcome__keys">{entry.keys}</kbd>
                                {/if}
                            </button>
                        </li>
                    {/each}
                </ul>
            </section>
            {#if recent.length > 0}
                <section class="welcome__recent">
                    <h2 class="welcome__heading">Recent</h2>
                    <ul class="welcome__list">
                        {#each recent as entry (entry.id)}
                            <li>
                                <button
                                    class="welcome__item"
                                    onclick={() => onOpenRecent(entry.id)}
                                    title={entry.detail}
                                >
                                    <span class="welcome__label">{entry.name}</span>
                                    <span class="welcome__detail">{entry.detail}</span>
                                </button>
                            </li>
                        {/each}
                    </ul>
                </section>
            {/if}
        </div>
    </div>
</div>

<style>
    .welcome {
        position: absolute;
        inset: 0;
        overflow-y: auto;
        background: var(--bg);
        color: var(--fg);
        user-select: none;
    }

    .welcome__content {
        width: min(760px, 100%);
        margin: 0 auto;
        padding: 64px 32px 48px;
    }

    .welcome__brand {
        display: flex;
        align-items: center;
        gap: 16px;
        margin-bottom: 40px;
    }

    .welcome__logo {
        width: 64px;
        height: 64px;
    }

    .welcome__title {
        margin: 0;
        font-size: 24px;
        font-weight: 500;
        line-height: 1.2;
    }

    .welcome__tagline {
        margin: 4px 0 0;
        color: var(--fg-muted);
    }

    .welcome__grid {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: 32px;
    }

    .welcome__recent {
        grid-column: 1 / -1;
    }

    @media (max-width: 680px) {
        .welcome__grid {
            grid-template-columns: 1fr;
        }

        .welcome__content {
            padding: 40px 24px 32px;
        }
    }

    .welcome__heading {
        margin: 0 0 8px;
        font-size: var(--ui-font-size);
        font-weight: 600;
        color: var(--fg-muted);
    }

    .welcome__list {
        margin: 0;
        padding: 0;
        list-style: none;
    }

    .welcome__item {
        display: flex;
        align-items: baseline;
        gap: 12px;
        width: 100%;
        padding: 5px 8px;
        border: none;
        border-radius: 4px;
        background: transparent;
        color: var(--fg);
        font: inherit;
        text-align: left;
        cursor: pointer;
    }

    .welcome__item:hover {
        background: var(--bg-hover);
    }

    .welcome__item:focus-visible {
        outline: 2px solid var(--fg-muted);
        outline-offset: -2px;
    }

    .welcome__label {
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .welcome__keys {
        margin-left: auto;
        flex: 0 0 auto;
        padding: 1px 5px;
        border: 1px solid var(--border);
        border-radius: 4px;
        background: var(--bg-raised);
        color: var(--fg-muted);
        font-family: inherit;
        font-size: var(--ui-font-small);
    }

    .welcome__detail {
        margin-left: auto;
        flex: 0 1 auto;
        overflow: hidden;
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
        text-overflow: ellipsis;
        white-space: nowrap;
    }
</style>
