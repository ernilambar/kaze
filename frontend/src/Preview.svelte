<script lang="ts">
    import { Browser } from '@wailsio/runtime';
    import { renderMarkdown } from './markdown';
    import { svgDataUrl, type PreviewKind } from './preview';

    interface Props {
        kind: PreviewKind;
        // content is the buffer as the editor holds it, so unsaved edits are
        // what the preview shows.
        content: string;
        name: string;
    }

    let { kind, content, name }: Props = $props();

    let html = $derived(kind === 'markdown' ? renderMarkdown(content) : '');
    let src = $derived(kind === 'svg' ? svgDataUrl(content) : '');

    // A link must never navigate the webview: that would unload the editor and
    // every unsaved buffer with it. Web and mail links open in the default
    // app; anything else is ignored.
    function onPreviewClick(event: MouseEvent): void {
        const target = event.target instanceof Element ? event.target.closest('a') : null;
        if (target === null) {
            return;
        }
        event.preventDefault();
        const href = target.getAttribute('href') ?? '';
        if (/^(https?|mailto):/i.test(href)) {
            void Browser.OpenURL(href);
        }
    }
</script>

{#if kind === 'svg'}
    <div class="preview preview--svg">
        <img class="preview__image" {src} alt={name} draggable="false" />
    </div>
{:else}
    <div class="preview">
        <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
        <article class="preview__page" onclick={onPreviewClick}>{@html html}</article>
    </div>
{/if}

<style>
    .preview {
        position: absolute;
        inset: 0;
        overflow: auto;
        background: var(--bg);
    }

    .preview--svg {
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 16px;
    }

    .preview__image {
        max-width: 100%;
        max-height: 100%;
        display: block;
        -webkit-user-select: none;
        user-select: none;
        -webkit-user-drag: none;
    }

    .preview__page {
        max-width: 860px;
        margin: 0 auto;
        padding: 24px 32px 48px;
        line-height: 1.6;
        color: var(--fg);
        overflow-wrap: break-word;
    }

    /* The rendered Markdown is inserted as HTML, so its elements are styled
       through :global; Svelte cannot scope content it never compiled. */
    .preview__page :global(h1),
    .preview__page :global(h2),
    .preview__page :global(h3),
    .preview__page :global(h4),
    .preview__page :global(h5),
    .preview__page :global(h6) {
        margin: 1.4em 0 0.5em;
        line-height: 1.25;
        font-weight: 600;
    }

    .preview__page :global(h1) {
        font-size: 2em;
        padding-bottom: 0.3em;
        border-bottom: 1px solid var(--border);
    }

    .preview__page :global(h2) {
        font-size: 1.5em;
        padding-bottom: 0.3em;
        border-bottom: 1px solid var(--border);
    }

    .preview__page :global(h3) {
        font-size: 1.25em;
    }

    .preview__page :global(p) {
        margin: 0.8em 0;
    }

    .preview__page :global(a) {
        color: var(--link, #4daafc);
        text-decoration: none;
    }

    .preview__page :global(a:hover) {
        text-decoration: underline;
    }

    .preview__page :global(code) {
        padding: 0.15em 0.35em;
        border-radius: 4px;
        background: var(--bg-raised);
        font-family: Menlo, monospace;
        font-size: 0.9em;
    }

    .preview__page :global(pre) {
        padding: 12px 16px;
        border: 1px solid var(--border);
        border-radius: 6px;
        background: var(--bg-sidebar);
        overflow-x: auto;
    }

    .preview__page :global(pre code) {
        padding: 0;
        background: none;
        font-size: 0.9em;
    }

    .preview__page :global(blockquote) {
        margin: 0.8em 0;
        padding: 0.2em 1em;
        border-left: 3px solid var(--border);
        color: var(--fg-muted);
    }

    .preview__page :global(ul),
    .preview__page :global(ol) {
        margin: 0.8em 0;
        padding-left: 1.5em;
    }

    .preview__page :global(li) {
        margin: 0.25em 0;
    }

    .preview__page :global(table) {
        border-collapse: collapse;
        margin: 1em 0;
    }

    .preview__page :global(th),
    .preview__page :global(td) {
        padding: 6px 12px;
        border: 1px solid var(--border);
        text-align: left;
    }

    .preview__page :global(th) {
        background: var(--bg-raised);
    }

    .preview__page :global(hr) {
        border: 0;
        border-top: 1px solid var(--border);
        margin: 1.5em 0;
    }

    .preview__page :global(img) {
        max-width: 100%;
    }
</style>
