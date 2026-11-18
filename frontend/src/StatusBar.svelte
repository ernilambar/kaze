<script lang="ts">
    import type { DocumentEOL } from './document';
    import { formatCursorPosition, lineEndingLabel } from './status';
    import type { SidebarView } from './git';

interface Props {
    line: number;
    column: number;
    // selectionLength is the number of selected characters; zero hides it.
    selectionLength?: number;
    language: string;
    // eol is empty for a non-text tab, which has no line ending to switch.
    eol?: DocumentEOL | '';
    // invalidUTF8 flags a file that was not valid UTF-8 when it was read.
    invalidUTF8?: boolean;
    // indentation is the active buffer's indent style and width, formatted
    // for display; empty for a tab with none to configure.
    indentation?: string;
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
    // onToggleEol switches the active buffer between LF and CRLF.
    onToggleEol?: () => void;
    // onPickIndentation opens the indentation list for the active buffer.
    onPickIndentation?: () => void;
    // onChangeLanguage opens the language mode list for the active buffer.
    onChangeLanguage?: () => void;
}

let {
    line,
    column,
    selectionLength = 0,
    language,
    eol = '',
    invalidUTF8 = false,
    indentation = '',
    notice = '',
    imageSize = '',
    branch = '',
    sidebarView = 'explorer',
    changesCount = 0,
    onSwitchView,
    onToggleEol,
    onPickIndentation,
    onChangeLanguage,
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
                aria-label="Explorer"
                title="Explorer"
                onclick={() => onSwitchView?.('explorer')}
            >
                <svg aria-hidden="true" height="12" viewBox="0 0 12 12" width="12">
                    <path
                        d="M1.5 3.6c0-.6.4-1 1-1h2.1l1.1 1.2h3.8c.6 0 1 .4 1 1v4.3c0 .6-.4 1-1 1h-7c-.6 0-1-.4-1-1z"
                        fill="none"
                        stroke="currentColor"
                        stroke-linejoin="round"
                        stroke-width="1.2"
                    />
                </svg>
            </button>
            <button
                class="statusbar__view"
                class:statusbar__view--active={sidebarView === 'changes'}
                role="tab"
                aria-selected={sidebarView === 'changes'}
                aria-label="Changes"
                title="Changes"
                onclick={() => onSwitchView?.('changes')}
            >
                <svg aria-hidden="true" height="12" viewBox="0 0 12 12" width="12">
                    <path
                        d="M3 1.5V7.5M9 4.4a4.6 4.6 0 0 1-4.6 4.6"
                        fill="none"
                        stroke="currentColor"
                        stroke-linecap="round"
                        stroke-width="1.1"
                    />
                    <circle cx="9" cy="3" fill="none" r="1.4" stroke="currentColor" stroke-width="1.1" />
                    <circle cx="3" cy="9" fill="none" r="1.4" stroke="currentColor" stroke-width="1.1" />
                </svg>
                {#if changesCount > 0}<span class="statusbar__count">{changesCount}</span>{/if}
            </button>
        </div>
        <span class="statusbar__position">{formatCursorPosition(line, column, selectionLength)}</span>
    </div>
    {#if notice !== ''}
        <span class="statusbar__notice">{notice}</span>
    {/if}
    <div class="statusbar__end">
        {#if invalidUTF8}
            <span
                class="statusbar__encoding"
                title="This file is not valid UTF-8. Saving replaces the bytes that could not be decoded with U+FFFD."
            >Invalid UTF-8</span>
        {/if}
        {#if branch !== ''}
            <span class="statusbar__branch" title="Current branch">{branch}</span>
        {/if}
        {#if imageSize !== ''}
            <span class="statusbar__image">{imageSize}</span>
        {/if}
        {#if indentation !== '' && onPickIndentation}
            <button
                class="statusbar__action statusbar__indent"
                type="button"
                title="Select Indentation"
                onclick={() => onPickIndentation?.()}
            >{indentation}</button>
        {/if}
        {#if eol !== ''}
            <button
                class="statusbar__action statusbar__eol"
                type="button"
                title="Select End of Line Sequence"
                onclick={() => onToggleEol?.()}
            >{lineEndingLabel(eol)}</button>
        {/if}
        {#if onChangeLanguage}
            <button
                class="statusbar__action statusbar__language"
                type="button"
                title="Select Language Mode"
                onclick={() => onChangeLanguage?.()}
            >{language}</button>
        {:else}
            <span class="statusbar__language">{language}</span>
        {/if}
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

    .statusbar__encoding {
        white-space: nowrap;
        color: var(--fg-danger);
    }

    .statusbar__action {
        padding: 0;
        border: none;
        background: none;
        color: inherit;
        font: inherit;
        white-space: nowrap;
        cursor: pointer;
    }

    .statusbar__action:hover {
        color: var(--fg);
    }
</style>
