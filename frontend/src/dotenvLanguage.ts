import type * as monaco from 'monaco-editor';

// DOTENV_LANGUAGE_ID is the id Monaco registers, and the one the status bar and
// Change Language Mode list read back from the registry.
export const DOTENV_LANGUAGE_ID = 'dotenv';

// DOTENV_EXTENSIONS is what Auto Detect matches for `name.env` files. A bare
// `.env` has no stem, and `.env.local` would resolve to `.local`, so the
// registry's extension list cannot describe either one; DOTENV_NAME covers
// them.
export const DOTENV_EXTENSIONS = ['.env'];

// DOTENV_NAME is the filename Auto Detect treats as dotenv: `.env` and any
// `.env.<stage>`. It is checked before the registry index, matching what every
// dotenv tool recognises.
export const DOTENV_NAME = /^\.env(?:\..+)?$/i;

// dotenvConf gives Toggle Line Comment the file's own comment marker.
export const dotenvConf: monaco.languages.LanguageConfiguration = {
    comments: { lineComment: '#' },
};

// A `$VAR` or `${VAR}` reference, including a `${VAR:-default}` default.
// Unquoted and double-quoted values expand it; single-quoted values do not.
const VARIABLE = /\$\{[^}\n]*\}|\$[A-Za-z_][A-Za-z0-9_]*/;

// dotenvLanguage follows the grammar dotenv parsers share: one `KEY=VALUE`
// assignment per line, optional `export`, `#` comments, and three value forms.
// Tokens are the ones syntaxRules in theme.ts styles, so every theme colors it.
export const dotenvLanguage: monaco.languages.IMonarchLanguage = {
    // The lexer needs the newline to reset value and quote states per line;
    // without it a state left at the end of one line colours the next one.
    includeLF: true,
    tokenizer: {
        root: [
            [/\s+/, ''],
            // Only a `#` that starts a token is a comment, so `KEY=bar#baz`
            // keeps the `#` inside the value while `KEY=bar # baz` comments.
            [/#.*(?=\n|$)/, 'comment'],
            [/export(?=\s)/, 'keyword'],
            [/[A-Za-z_][A-Za-z0-9_.-]*/, 'key'],
            [/=/, 'operator', '@value'],
            [/[^\s=#]+/, 'invalid'],
        ],
        value: [
            [/\n/, '', '@pop'],
            [/'/, 'string', '@singleQuoted'],
            [/"/, 'string', '@doubleQuoted'],
            [/\s+/, ''],
            [/#.*(?=\n|$)/, 'comment'],
            [VARIABLE, 'variable'],
            // A value chunk may contain `#`; it just cannot start with one.
            [/[^\s'"$#][^\s'"$]*/, 'string'],
            [/\$/, 'string'],
        ],
        singleQuoted: [
            // Quotes are single-line, so an unterminated one ends at the
            // newline instead of colouring the rest of the file.
            [/\n/, '', '@pop'],
            [/'/, 'string', '@pop'],
            [/[^']+/, 'string'],
        ],
        doubleQuoted: [
            [/\n/, '', '@pop'],
            [/"/, 'string', '@pop'],
            [/\\[\\'"nrt$]/, 'string'],
            [VARIABLE, 'variable'],
            [/[^"\\$]+/, 'string'],
            [/[$\\]/, 'string'],
        ],
    },
};
