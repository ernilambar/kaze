import type * as monaco from 'monaco-editor';

// IGNORE_LANGUAGE_ID is the id Monaco registers, and the one the status bar and
// Change Language Mode list read back from the registry.
export const IGNORE_LANGUAGE_ID = 'ignore';

// IGNORE_EXTENSIONS covers `name.ignore` files. The dotfiles below have no
// extension Monaco could index, so IGNORE_NAME covers them.
export const IGNORE_EXTENSIONS = ['.ignore'];

// IGNORE_NAME matches the dotfiles that share the gitignore syntax: any
// `.<tool>ignore` name, its `_global` variant, and the bare `.ignore`. A
// `.bak` file is a byte copy of any original and stays plain text.
export const IGNORE_NAME =
    /^(?!.*\.bak$)(?:\.[\w.-]*ignore|\.[\w.-]*ignore[-_]global)$/i;

// ignoreLanguage follows gitignore syntax, which every tool using these names
// shares: `#` comments at the start of a line, a leading `!` that re-includes
// a path, `/` separators and directory markers, `*`/`?`/`**` wildcards and
// `[...]` classes. Literal path text stays on the default foreground; comments
// and the syntax characters carry the color. Token names are the ones
// syntaxRules in theme.ts styles, so every theme colors it.
export const ignoreLanguage: monaco.languages.IMonarchLanguage = {
    tokenizer: {
        root: [
            // A `#` only starts a comment at the start of a line; anywhere else
            // it is part of the pattern and stays literal.
            [/^[ \t]*#.*$/, 'comment'],
            [/^!/, 'keyword'],
            // A backslash makes the next character literal, including the
            // `#`, `!` and wildcard characters above.
            [/\\[\\ !#*?[\]{}]/, 'string.escape'],
            [/\*\*/, 'regexp'],
            [/[*?]/, 'regexp'],
            [/\[(?:\\.|[^\]\\])*\]/, 'regexp'],
            // Unbalanced `[` is still a class start to the reader.
            [/\[/, 'regexp'],
            // Both the separator and the trailing directory marker.
            [/\//, 'delimiter'],
            // Everything else is literal path text and keeps the default color.
            [/[^\\*?\[\]\/]+/, ''],
        ],
    },
};
