<script lang="ts">
    import { System } from '@wailsio/runtime';

    interface Props {
        appName: string;
        fileName: string;
        folderLabel?: string;
    }

    let { appName, fileName, folderLabel = '' }: Props = $props();

    // macOS hides its native title bar and floats the traffic lights over the
    // content, so this bar leaves room for them. Elsewhere the native bar stays
    // and no inset is needed.
    const mac = System.IsMac();
</script>

<header class="titlebar" class:titlebar--mac={mac}>
    <span class="titlebar__app">{appName}</span>
    <span class="titlebar__file">{fileName}</span>
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
