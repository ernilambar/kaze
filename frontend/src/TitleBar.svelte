<script lang="ts">
    interface Props {
        appName: string;
        filePath: string;
        folderLabel?: string;
    }

    let { appName, filePath, folderLabel = '' }: Props = $props();

    // macOS hides its native title bar and floats the traffic lights over the
    // content, so this bar leaves room for them. Elsewhere the native bar stays
    // and no inset is needed. System.IsMac() is not used: Wails injects
    // window._wails.environment only after navigation finishes, so it can
    // still be unset when this component first renders.
    const mac = navigator.userAgent.includes('Macintosh');
</script>

<header class="titlebar" class:titlebar--mac={mac}>
    <span class="titlebar__app">{appName}</span>
    <span class="titlebar__file" title={filePath}>{filePath}</span>
    {#if folderLabel !== ''}
        <span class="titlebar__folder">{folderLabel}</span>
    {/if}
</header>

<style>
    .titlebar {
        display: flex;
        align-items: center;
        gap: 8px;
        flex: 0 0 auto;
        height: 38px;
        padding: 0 12px;
        background: var(--bg-raised);
        border-bottom: 1px solid var(--border);
        -webkit-user-select: none;
        user-select: none;
        /* No native title bar covers this strip, so dragging it moves the
           window and a double click zooms it. */
        --wails-draggable: drag;
    }

    .titlebar--mac {
        /* The traffic lights are centred about 16px from the top. The bottom
           border and the glyphs' cap height sit the text above the box's
           midline, so the top padding lowers it onto the lights' centre. */
        height: 32px;
        padding-top: 2px;
        padding-left: 80px;
    }

    .titlebar__app {
        font-weight: 600;
    }

    .titlebar__file {
        color: var(--fg-muted);
        /* A full path is longer than a name, so it truncates to fit instead of
           pushing the workspace label off the bar. The title attribute keeps
           the whole path one hover away. */
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .titlebar__file::before {
        content: "—";
        margin-right: 8px;
    }

    .titlebar__folder {
        margin-left: auto;
        color: var(--fg-muted);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
</style>
