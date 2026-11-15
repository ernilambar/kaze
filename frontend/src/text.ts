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
