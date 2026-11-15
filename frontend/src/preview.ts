// PreviewKind is the surface Preview.svelte knows how to draw.
export type PreviewKind = 'markdown' | 'svg';

const MARKDOWN_EXTENSIONS = [
    '.md',
    '.markdown',
    '.mdown',
    '.mkdn',
    '.mkd',
    '.mdwn',
    '.mdtxt',
    '.mdtext',
];

// previewKindFor picks the preview a path supports, or '' when there is none.
export function previewKindFor(path: string): PreviewKind | '' {
    const lower = path.toLowerCase();
    if (lower.endsWith('.svg')) {
        return 'svg';
    }
    return MARKDOWN_EXTENSIONS.some((extension) => lower.endsWith(extension))
        ? 'markdown'
        : '';
}

// svgDataUrl embeds an SVG document for an <img>, which keeps the preview
// sandboxed: scripts inside the file never run.
export function svgDataUrl(svg: string): string {
    return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`;
}
