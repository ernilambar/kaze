<script lang="ts">
    interface Props {
        appName: string;
        fileName: string;
        folder?: string;
    }

    let { appName, fileName, folder = '' }: Props = $props();
</script>

<header class="titlebar">
    <span class="titlebar__app">{appName}</span>
    <span class="titlebar__file">{fileName}</span>
    {#if folder !== ''}
        <span class="titlebar__folder">{folder}</span>
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
