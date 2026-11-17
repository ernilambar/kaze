// trimTrailingWhitespace removes spaces and tabs at the end of every line and
// leaves the line endings untouched.
export function trimTrailingWhitespace(text: string): string {
    return text.replace(/[ \t]+$/gm, '');
}

// appendFinalNewline gives text the single newline ensure_final_newline
// promises. An empty buffer stays empty: a file with no lines has no final line
// to end.
export function appendFinalNewline(text: string): string {
    if (text === '' || text.endsWith('\n')) {
        return text;
    }
    return `${text}\n`;
}

// LineTransform names one whole-line command. The ids are the command palette's
// ids as well.
export type LineTransform =
    | 'sort-lines-ascending'
    | 'sort-lines-descending'
    | 'delete-duplicate-lines'
    | 'reverse-lines';

// mapLines rewrites the lines of text and keeps the final newline a file may
// end with. Splitting on '\n' yields an empty trailing element for that
// newline, and it must not take part in a sort or a reverse.
function mapLines(text: string, transform: (lines: string[]) => string[]): string {
    const finalNewline = text.endsWith('\n');
    const lines = text.split('\n');
    if (finalNewline) {
        lines.pop();
    }
    return `${transform(lines).join('\n')}${finalNewline ? '\n' : ''}`;
}

// transformLines applies one of the line commands to text. Sorting compares
// code units and is stable, so equal lines keep their order.
export function transformLines(text: string, transform: LineTransform): string {
    switch (transform) {
        case 'sort-lines-ascending':
            return mapLines(text, (lines) =>
                [...lines].sort((a, b) => (a < b ? -1 : a > b ? 1 : 0)),
            );
        case 'sort-lines-descending':
            return mapLines(text, (lines) =>
                [...lines].sort((a, b) => (a < b ? 1 : a > b ? -1 : 0)),
            );
        case 'delete-duplicate-lines':
            return mapLines(text, (lines) => [...new Set(lines)]);
        case 'reverse-lines':
            return mapLines(text, (lines) => [...lines].reverse());
    }
}
