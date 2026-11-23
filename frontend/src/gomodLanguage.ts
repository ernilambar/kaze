import type * as monaco from 'monaco-editor';

// GOMOD_LANGUAGE_ID is the id Monaco registers, and the one the status bar and
// Change Language Mode list read back from the registry.
export const GOMOD_LANGUAGE_ID = 'gomod';

// GOMOD_NAMES are the filenames Auto Detect treats as Go module files. The
// registry indexes these directly: `go.mod` and `go.work` are exact names, and
// the `.mod` extension must stay unclaimed because other ecosystems use it.
export const GOMOD_NAMES = ['go.mod', 'go.work'] as const;

// gomodConf gives Toggle Line Comment the file's own marker and pairs the
// brackets a `require (...)` block and a `retract [...]` range nest. Go module
// files accept only `//` comments; the go command rejects `/* */`, so no block
// comment is configured here. Strings are double-quoted only; a back quote is
// not a string delimiter.
export const gomodConf: monaco.languages.LanguageConfiguration = {
    comments: { lineComment: '//' },
    brackets: [
        ['(', ')'],
        ['[', ']'],
    ],
    autoClosingPairs: [
        { open: '(', close: ')' },
        { open: '[', close: ']' },
        { open: '"', close: '"', notIn: ['string'] },
    ],
    surroundingPairs: [
        { open: '(', close: ')' },
        { open: '[', close: ']' },
        { open: '"', close: '"' },
    ],
};

// A version as the go command writes it: `v1.2.3` in require lines, the bare
// `1.22` after the `go` directive, a pseudo-version like
// `v0.0.0-20200101000000-abcdefabcdef`, and a `+incompatible` suffix.
const VERSION = /v?\d+\.\d+(?:\.\d+)?(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?/;

// A toolchain name as the `toolchain` directive writes it: the version carries
// the `go` prefix, optionally with a prerelease tag (`go1.22rc1`).
const TOOLCHAIN_VERSION = /go\d+\.\d+(?:\.\d+)?(?:[a-z]+\d+)?(?:[-+][0-9A-Za-z.-]+)?/;

// The directives the go and gofmt parsers know. `go` is the language-version
// directive; it must be followed by whitespace (or end of line) so module
// paths like `gopkg.in/yaml.v3` are not split at their first two letters.
const DIRECTIVE = /(?:module|go|toolchain|godebug|require|exclude|replace|retract|use)(?=[\s([{])/;

// gomodLanguage is a Monarch grammar for go.mod and go.work: directives,
// parenthesised and bracketed blocks, `replace old => new` arrows, module
// paths, versions and double-quoted strings, with the go command's own `//`
// comments. The token names are the ones syntaxRules in theme.ts styles, so
// every theme colors it: directives are `keyword`, versions `number` and paths
// `key`.
export const gomodLanguage: monaco.languages.IMonarchLanguage = {
    defaultToken: '',
    tokenPostfix: '.gomod',
    includeLF: true,
    tokenizer: {
        root: [
            [/[ \t]+/, ''],
            [/\n/, ''],
            [/\/\/.*(?=\n|$)/, 'comment'],
            [DIRECTIVE, 'keyword'],
            [/"/, 'string', '@doubleQuoted'],
            [/[[\]{}()]/, 'delimiter.bracket'],
            [/,/, 'delimiter'],
            [/=>/, 'operator'],
            [TOOLCHAIN_VERSION, 'number'],
            [VERSION, 'number'],
            // A bare token: a module path, `./local` replacement target or
            // plain identifier. Slashes are ordinary characters here, but a
            // `//` ends the token, so a comment written without a space before
            // it still starts one.
            [/(?:(?!\/\/)[^\s()[\]{}"`])+/, 'key'],
        ],
        doubleQuoted: [
            // Strings are single-line in practice, so an unterminated one
            // ends at the newline instead of coloring the rest of the file.
            [/\n/, '', '@pop'],
            // Go escaping: a backslash keeps the quote it precedes inside the
            // string instead of closing it.
            [/\\./, 'string'],
            [/"/, 'string', '@pop'],
            [/[^"\\\n]+/, 'string'],
            [/\\/, 'string'],
        ],
    },
};
