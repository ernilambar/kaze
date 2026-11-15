// The app previews Markdown without adding a dependency, so this renderer covers
// the block and inline forms documents use. All text comes from the user's
// buffer, so it is HTML-escaped and links are limited to safe schemes before
// the result reaches {@html}.

const ESCAPES: Record<string, string> = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
};

export function escapeHtml(text: string): string {
    return text.replace(/[&<>"']/g, (character) => ESCAPES[character] ?? character);
}

// safeUrl drops schemes the preview must not follow. Relative paths and
// fragments stay: they are how a repository's own files are linked.
function safeUrl(url: string): string {
    const trimmed = url.trim();
    if (trimmed === '') {
        return '';
    }
    if (/^[a-z][a-z0-9+.-]*:/i.test(trimmed)) {
        return /^(https?|mailto):/i.test(trimmed) ? trimmed : '';
    }
    return trimmed;
}

// The alternatives, in order: code span, image, link, strong, strong, emphasis,
// emphasis, strikethrough. Capture groups follow the same order.
const INLINE = new RegExp(
    [
        '(`+)([\\s\\S]*?)\\1',
        '!\\[([^\\]]*)\\]\\(([^)\\s]+)(?:\\s+"([^"]*)")?\\)',
        '\\[([^\\]]*)\\]\\(([^)\\s]+)(?:\\s+"([^"]*)")?\\)',
        '\\*\\*([\\s\\S]+?)\\*\\*',
        '(?<![\\w])__([\\s\\S]+?)__(?![\\w])',
        '\\*([^*\\n]+?)\\*',
        '(?<![\\w])_([^_\\n]+?)_(?![\\w])',
        '~~([\\s\\S]+?)~~',
    ].join('|'),
    'g',
);

const FENCE = /^\s*(`{3,}|~{3,})\s*([\w+-]*)\s*$/;
const HEADING = /^(#{1,6})\s+(.+?)\s*#*\s*$/;
const RULE = /^\s{0,3}([-*_])(?:\s*\1){2,}\s*$/;
const LIST_ITEM = /^\s*([-*+]|\d+[.)])\s+(.*)$/;
const QUOTE = /^\s*>\s?(.*)$/;
const TABLE_SEPARATOR = /^\s*\|?\s*:?-{2,}:?\s*(?:\|\s*:?-{2,}:?\s*)*\|?\s*$/;

function titleAttribute(title: string | undefined): string {
    return title === undefined || title === '' ? '' : ` title="${escapeHtml(title)}"`;
}

function renderInline(text: string): string {
    let html = '';
    let last = 0;
    for (const match of text.matchAll(INLINE)) {
        const start = match.index ?? 0;
        html += escapeHtml(text.slice(last, start));
        last = start + match[0].length;

        if (match[1] !== undefined) {
            html += `<code>${escapeHtml(match[2])}</code>`;
        } else if (match[3] !== undefined) {
            const url = safeUrl(match[4]);
            html +=
                url === ''
                    ? escapeHtml(match[3])
                    : `<img src="${escapeHtml(url)}" alt="${escapeHtml(match[3])}"${titleAttribute(match[5])}>`;
        } else if (match[6] !== undefined) {
            const url = safeUrl(match[7]);
            html +=
                url === ''
                    ? escapeHtml(match[6])
                    : `<a href="${escapeHtml(url)}"${titleAttribute(match[8])}>${renderInline(match[6])}</a>`;
        } else if (match[9] !== undefined) {
            html += `<strong>${renderInline(match[9])}</strong>`;
        } else if (match[10] !== undefined) {
            html += `<strong>${renderInline(match[10])}</strong>`;
        } else if (match[11] !== undefined) {
            html += `<em>${renderInline(match[11])}</em>`;
        } else if (match[12] !== undefined) {
            html += `<em>${renderInline(match[12])}</em>`;
        } else if (match[13] !== undefined) {
            html += `<del>${renderInline(match[13])}</del>`;
        }
    }
    return html + escapeHtml(text.slice(last));
}

function splitRow(row: string): string[] {
    return row
        .trim()
        .replace(/^\|/, '')
        .replace(/\|$/, '')
        .split('|')
        .map((cell) => cell.trim());
}

function renderListItem(text: string): string {
    const task = /^\[([ xX])\]\s+(.*)$/.exec(text);
    if (task === null) {
        return renderInline(text);
    }
    const checked = task[1].toLowerCase() === 'x' ? ' checked' : '';
    return `<input type="checkbox" disabled${checked}> ${renderInline(task[2])}`;
}

function startsBlock(line: string): boolean {
    return (
        FENCE.test(line) ||
        HEADING.test(line) ||
        RULE.test(line) ||
        QUOTE.test(line) ||
        LIST_ITEM.test(line) ||
        line.trim().startsWith('|')
    );
}

export function renderMarkdown(source: string): string {
    const lines = source.replace(/\r\n?/g, '\n').split('\n');
    const html: string[] = [];
    let at = 0;

    while (at < lines.length) {
        const line = lines[at] ?? '';

        if (line.trim() === '') {
            at++;
            continue;
        }

        const fence = FENCE.exec(line);
        if (fence !== null) {
            const marker = fence[1];
            const closing = new RegExp(`^\\s*${marker.charAt(0)}{${marker.length},}\\s*$`);
            const code: string[] = [];
            at++;
            while (at < lines.length) {
                const candidate = lines[at] ?? '';
                at++;
                if (closing.test(candidate)) {
                    break;
                }
                code.push(candidate);
            }
            const language = fence[2] === '' ? '' : ` class="language-${escapeHtml(fence[2])}"`;
            html.push(`<pre><code${language}>${escapeHtml(code.join('\n'))}</code></pre>`);
            continue;
        }

        const heading = HEADING.exec(line);
        if (heading !== null) {
            const level = heading[1].length;
            html.push(`<h${level}>${renderInline(heading[2])}</h${level}>`);
            at++;
            continue;
        }

        if (RULE.test(line)) {
            html.push('<hr>');
            at++;
            continue;
        }

        if (QUOTE.test(line)) {
            const quoted: string[] = [];
            while (at < lines.length) {
                const match = QUOTE.exec(lines[at] ?? '');
                if (match === null) {
                    break;
                }
                quoted.push(match[1]);
                at++;
            }
            html.push(`<blockquote>${renderMarkdown(quoted.join('\n'))}</blockquote>`);
            continue;
        }

        if (
            line.includes('|') &&
            at + 1 < lines.length &&
            TABLE_SEPARATOR.test(lines[at + 1] ?? '')
        ) {
            const headers = splitRow(line);
            at += 2;
            const rows: string[][] = [];
            while (at < lines.length) {
                const row = lines[at] ?? '';
                if (row.trim() === '' || !row.includes('|')) {
                    break;
                }
                rows.push(splitRow(row));
                at++;
            }
            const head = headers.map((cell) => `<th>${renderInline(cell)}</th>`).join('');
            const body = rows
                .map(
                    (row) =>
                        `<tr>${headers
                            .map((_, cell) => `<td>${renderInline(row[cell] ?? '')}</td>`)
                            .join('')}</tr>`,
                )
                .join('');
            html.push(`<table><thead><tr>${head}</tr></thead><tbody>${body}</tbody></table>`);
            continue;
        }

        const firstItem = LIST_ITEM.exec(line);
        if (firstItem !== null) {
            const ordered = /\d/.test(firstItem[1]);
            const items: string[] = [];
            while (at < lines.length) {
                const item = LIST_ITEM.exec(lines[at] ?? '');
                if (item === null) {
                    break;
                }
                items.push(`<li>${renderListItem(item[2])}</li>`);
                at++;
            }
            html.push(ordered ? `<ol>${items.join('')}</ol>` : `<ul>${items.join('')}</ul>`);
            continue;
        }

        const paragraph: string[] = [];
        while (at < lines.length) {
            const current = lines[at] ?? '';
            if (current.trim() === '' || (paragraph.length > 0 && startsBlock(current))) {
                break;
            }
            paragraph.push(current.trim());
            at++;
        }
        html.push(`<p>${paragraph.map(renderInline).join(' ')}</p>`);
    }

    return html.join('');
}
