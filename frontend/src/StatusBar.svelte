<script lang="ts">
    import { formatCursorPosition } from './status';
    import type { SidebarView } from './git';

interface Props {
    line: number;
    column: number;
    language: string;
    folderLabel?: string;
    notice?: string;
    // imageSize is the natural pixel size of the image tab, formatted for
    // display; empty for text tabs.
    imageSize?: string;
    // branch is the active repository's branch and ahead/behind counts.
    branch?: string;
    // sidebarView and changesCount drive the Explorer/Changes switch.
    sidebarView?: SidebarView;
    changesCount?: number;
    onSwitchView?: (view: SidebarView) => void;
}

let {
    line,
    column,
    language,
    folderLabel = '',
    notice = '',
    imageSize = '',
    branch = '',
    sidebarView = 'explorer',
    changesCount = 0,
    onSwitchView,
}: Props = $props();
</script>

<footer class="statusbar">
    <div class="statusbar__start">
        <div class="statusbar__views" role="tablist" aria-label="Sidebar view">
            <button
                class="statusbar__view"
                class:statusbar__view--active={sidebarView === 'explorer'}
                role="tab"
                aria-selected={sidebarView === 'explorer'}
                onclick={() => onSwitchView?.('explorer')}
            >Explorer</button>
            <button
                class="statusbar__view"
                class:statusbar__view--active={sidebarView === 'changes'}
                role="tab"
                aria-selected={sidebarView === 'changes'}
                onclick={() => onSwitchView?.('changes')}
            >Changes{#if changesCount > 0}<span class="statusbar__count">{changesCount}</span>{/if}</button>
        </div>
        <span class="statusbar__position">{formatCursorPosition(line, column)}</span>
    </div>
    {#if notice !== ''}
        <span class="statusbar__notice">{notice}</span>
    {/if}
    <div class="statusbar__end">
        {#if branch !== ''}
            <span class="statusbar__branch" title="Current branch">{branch}</span>
        {/if}
        {#if imageSize !== ''}
            <span class="statusbar__image">{imageSize}</span>
        {/if}
        {#if folderLabel !== ''}
            <span class="statusbar__folder">{folderLabel}</span>
        {/if}
        <span class="statusbar__language">{language}</span>
    </div>
</footer>

<style>
    .statusbar {
        display: flex;
        align-items: center;
        gap: 12px;
        flex: 0 0 auto;
        height: 24px;
        padding: 0 12px;
        line-height: 24px;
        background: var(--bg-raised);
        border-top: 1px solid var(--border);
        color: var(--fg-muted);
        user-select: none;
    }

    .statusbar__start {
        display: flex;
        align-items: center;
        gap: 12px;
        flex: 0 0 auto;
    }

    .statusbar__end {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-left: auto;
        min-width: 0;
    }

    .statusbar__views {
        display: inline-flex;
        align-items: center;
        padding: 1px;
        border: 1px solid var(--border);
        border-radius: 4px;
        background: var(--bg);
    }

    .statusbar__view {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        height: 18px;
        padding: 0 7px;
        border: none;
        border-radius: 3px;
        background: none;
        color: var(--fg-muted);
        font: inherit;
        font-size: var(--ui-font-small);
        cursor: pointer;
    }

    .statusbar__view:hover {
        color: var(--fg);
    }

    .statusbar__view--active {
        background: var(--bg-selected);
        color: var(--fg);
    }

    .statusbar__count {
        min-width: 14px;
        padding: 0 4px;
        border-radius: 7px;
        background: var(--bg-hover);
        color: var(--fg);
        font-size: var(--ui-font-tiny);
        line-height: 14px;
        text-align: center;
    }

    .statusbar__notice {
        min-width: 0;
        flex: 0 1 auto;
        color: var(--fg);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .statusbar__branch {
        white-space: nowrap;
        color: var(--fg);
    }

    .statusbar__image {
        white-space: nowrap;
    }

    .statusbar__folder {
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
</style>
