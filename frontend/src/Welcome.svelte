<script lang="ts">
    interface Entry {
        command: string;
        label: string;
        keys: string;
    }

    // The key labels mirror the default accelerators in menu.go; a custom
    // keybindings.json entry may override one.
    const ACTIONS: readonly Entry[] = [
        { command: 'new-buffer', label: 'New Untitled File', keys: '⌘N' },
        { command: 'open-file', label: 'Open File…', keys: '⌘O' },
        { command: 'add-folder', label: 'Add Folder…', keys: '⌃⌘O' },
    ];
</script>

<div class="welcome">
    <div class="welcome__content">
        <div class="welcome__logo" aria-hidden="true"></div>
        <ul class="welcome__list">
            {#each ACTIONS as entry (entry.command)}
                <li class="welcome__item">
                    <span class="welcome__label">{entry.label}</span>
                    <span class="welcome__keys">{entry.keys}</span>
                </li>
            {/each}
        </ul>
    </div>
</div>

<style>
    .welcome {
        position: absolute;
        inset: 0;
        display: flex;
        overflow-y: auto;
        background: var(--bg);
        color: var(--fg);
        -webkit-user-select: none;
        user-select: none;
    }

    .welcome__content {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 64px;
        margin: auto;
        padding: 40px 24px;
    }

    .welcome__logo {
        position: relative;
        width: 160px;
        height: 160px;
        border-radius: 23%;
        background: var(--bg-raised);
    }

    .welcome__logo::after {
        position: absolute;
        inset: 0;
        background: var(--fg-muted);
        content: '';
        -webkit-mask: url('./assets/glyph.png') center / 100% 100% no-repeat;
        mask: url('./assets/glyph.png') center / 100% 100% no-repeat;
    }

    .welcome__list {
        display: flex;
        flex-direction: column;
        gap: 2px;
        width: 320px;
        max-width: 100%;
        margin: 0;
        padding: 0;
        list-style: none;
    }

    .welcome__item {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
        padding: 8px 10px;
        color: var(--fg);
        font-size: var(--ui-font-size);
    }

    .welcome__label {
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .welcome__keys {
        flex: 0 0 auto;
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
    }
</style>
