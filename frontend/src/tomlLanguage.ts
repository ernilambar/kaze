import type * as monaco from 'monaco-editor';

// TOML_LANGUAGE_ID is the id Monaco registers, and the one the status bar and
// Change Language Mode list read back from the registry.
export const TOML_LANGUAGE_ID = 'toml';

// TOML_EXTENSIONS is what Auto Detect matches. Every TOML file carries one,
// including Cargo.toml, pyproject.toml and .cargo/config.toml, so the registry
// index needs no filename rule.
export const TOML_EXTENSIONS = ['.toml'];

// tomlConf gives Toggle Line Comment the file's own marker and pairs the
// brackets TOML nests. Only `#` starts a comment; a `;` is ordinary content.
export const tomlConf: monaco.languages.LanguageConfiguration = {
    comments: { lineComment: '#' },
    brackets: [
        ['[', ']'],
        ['{', '}'],
    ],
    autoClosingPairs: [
        { open: '[', close: ']' },
        { open: '{', close: '}' },
        { open: '"', close: '"' },
        { open: "'", close: "'" },
    ],
    surroundingPairs: [
        { open: '[', close: ']' },
        { open: '{', close: '}' },
        { open: '"', close: '"' },
        { open: "'", close: "'" },
    ],
};

// A date optionally followed by a time, and a time on its own: TOML's
// offset/local date-times and local dates and times. Both must win over the
// number rule below, which would otherwise stop at the first `-` or `:`.
const DATE = /\d{4}-\d{2}-\d{2}(?:[Tt]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:[Zz]|[+-]\d{2}:\d{2})?|\s\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:[Zz]|[+-]\d{2}:\d{2})?)?/;
const TIME = /\d{2}:\d{2}:\d{2}(?:\.\d+)?/;

// Integers (decimal, hex, octal, binary), floats with a fractional part or an
// exponent, and the signed `inf`/`nan` values. Underscores are allowed between
// digits, never at an edge.
const NUMBER = /[+-]?(?:0x[0-9A-Fa-f](?:_?[0-9A-Fa-f])*|0o[0-7](?:_?[0-7])*|0b[01](?:_?[01])*|(?:0|[1-9](?:_?\d)*)(?:\.\d(?:_?\d)*(?:[eE][+-]?\d(?:_?\d)*)?|[eE][+-]?\d(?:_?\d)*)?)/;

// tomlLanguage is a Monarch grammar for TOML 1.0: table headers, dotted and
// quoted keys, basic/literal strings (single and multi-line), arrays, inline
// tables, numbers, booleans and the four date-time forms. The token names are
// the ones syntaxRules in theme.ts styles, so every theme colors it: booleans
// are `constant` (the boolean capture) and table headers and keys are `key`.
export const tomlLanguage: monaco.languages.IMonarchLanguage = {
    defaultToken: '',
    tokenPostfix: '.toml',
    // The lexer needs the newlines to end a value and a table header. Without
    // it a state left at the end of one line would color the next one.
    includeLF: true,
    tokenizer: {
        // A key path (`a.b."c"`) lives in root: keys, dots and `=` share the
        // line, and a value leaves for its own state and pops back here.
        root: [
            [/\r?\n/, ''],
            [/[ \t]+/, ''],
            [/#[^\n]*/, 'comment'],
            // An array of tables before its single-table sibling.
            [/\[\[/, 'delimiter.bracket', '@header'],
            [/\[/, 'delimiter.bracket', '@header'],
            [/"/, 'string', '@basicString'],
            [/'/, 'string', '@literalString'],
            [/[A-Za-z0-9_-]+/, 'key'],
            [/\./, 'delimiter'],
            [/=/, 'delimiter', '@value'],
            [/[^\r\n]/, 'invalid'],
        ],
        // Table headers hold the same key path as root, closed by `]` or `]]`.
        header: [
            [/\r?\n/, '', '@pop'],
            [/[ \t]+/, ''],
            [/"/, 'string', '@basicString'],
            [/'/, 'string', '@literalString'],
            [/[A-Za-z0-9_-]+/, 'key'],
            [/\./, 'delimiter'],
            [/\]\]/, 'delimiter.bracket', '@pop'],
            [/\]/, 'delimiter.bracket', '@pop'],
            [/[^\r\n]/, 'invalid'],
        ],
        // A value ends at the newline, so `key = 1` cannot leak into the next
        // line; a trailing comment ends it early but colors as a comment.
        value: [
            [/\r?\n/, '', '@pop'],
            [/[ \t]+/, ''],
            [/#[^\n]*/, 'comment', '@pop'],
            { include: '@scalar' },
            [/[^\r\n]/, 'invalid'],
        ],
        // The literals every value position shares: arrays, inline tables and
        // inline-table values reuse these rules.
        scalar: [
            [/"""/, 'string', '@multilineBasic'],
            [/'''/, 'string', '@multilineLiteral'],
            [/"/, 'string', '@basicString'],
            [/'/, 'string', '@literalString'],
            [/\{/, 'delimiter.bracket', '@inlineTable'],
            [/\[/, 'delimiter.bracket', '@array'],
            [DATE, 'number'],
            [TIME, 'number'],
            [NUMBER, 'number'],
            [/(?:true|false)\b/, 'constant'],
            [/[+-]?(?:inf|nan)\b/, 'number'],
        ],
        // Arrays span lines and allow comments and a trailing comma.
        array: [
            [/\r?\n/, ''],
            [/[ \t]+/, ''],
            [/#[^\n]*/, 'comment'],
            [/\]/, 'delimiter.bracket', '@pop'],
            [/,/, 'delimiter'],
            { include: '@scalar' },
            [/[^\r\n]/, 'invalid'],
        ],
        // Inline tables hold `key = value` pairs. A `,` or `}` closes the
        // value first, then is rematched here as its own delimiter; newlines
        // stay inside, as TOML 1.1 allows.
        inlineTable: [
            [/\r?\n/, ''],
            [/[ \t]+/, ''],
            [/"/, 'string', '@basicString'],
            [/'/, 'string', '@literalString'],
            [/[A-Za-z0-9_-]+/, 'key'],
            [/\./, 'delimiter'],
            [/=/, 'delimiter', '@inlineValue'],
            [/,/, 'delimiter'],
            [/\}/, 'delimiter.bracket', '@pop'],
            [/[^\r\n]/, 'invalid'],
        ],
        inlineValue: [
            [/\r?\n/, '', '@pop'],
            [/[ \t]+/, ''],
            [/[,}]/, { token: '@rematch', next: '@pop' }],
            { include: '@scalar' },
            [/[^\r\n]/, 'invalid'],
        ],
        // Basic strings escape their quotes and cannot cross a newline; the
        // multi-line form below is a separate state.
        basicString: [
            [/[^"\\\n]+/, 'string'],
            [/\\./, 'string.escape'],
            [/"/, 'string', '@pop'],
            [/\r?\n/, 'string.invalid', '@pop'],
        ],
        // Literal strings take no escapes; only the closing quote matters.
        literalString: [
            [/[^'\n]+/, 'string'],
            [/'/, 'string', '@pop'],
            [/\r?\n/, 'string.invalid', '@pop'],
        ],
        // `"""..."""`: escapes work, and a backslash at the end of a line
        // trims the whitespace that follows.
        multilineBasic: [
            [/\\\r?\n/, 'string.escape'],
            [/\\./, 'string.escape'],
            [/"""/, 'string', '@pop'],
            [/[^"\\]+/, 'string'],
            [/[\\"]/, 'string'],
        ],
        // `'''...'''`: everything is content until the closing quotes, so a
        // run of one or two quotes stays inside the string.
        multilineLiteral: [
            [/'''/, 'string', '@pop'],
            [/[^']+/, 'string'],
            [/'/, 'string'],
        ],
    },
};
