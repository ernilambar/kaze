import type * as monaco from 'monaco-editor';

// DIFF_LANGUAGE_ID is the id Monaco registers. The read-only diff tabs use it
// directly, and Auto Detect resolves `.patch` and `.diff` files to it.
export const DIFF_LANGUAGE_ID = 'diff';

// DIFF_EXTENSIONS is what Auto Detect matches. `.patch` files are unified
// diffs as often as `.diff` files are, so both open highlighted.
export const DIFF_EXTENSIONS = ['.diff', '.patch'];

// The metadata that starts a file's section or describes how it changed. Git
// writes these before the `---`/`+++` header; `diff` covers the plain unified
// form too.
const FILE_META =
    /^(?:diff|index|new file mode|deleted file mode|old mode|new mode|similarity index|dissimilarity index|rename from|rename to|copy from|copy to|Binary files|GIT binary patch|literal|delta)\b.*$/;

// A hunk header, including the `@@@ ... @@@` form a combined (merge) diff uses.
const HUNK_HEADER = /^@@@?.*@@.*$/;

// diffLanguage is a Monarch grammar for unified diffs. File metadata and hunk
// headers get `keyword`, added lines `string` and removed lines `invalid`;
// context lines stay on the default foreground. The hunk state is what keeps a
// body line that begins with `---` or `+++` an addition or deletion instead of
// a file header. The token names are the ones syntaxRules in theme.ts styles,
// so every theme colors the diff without a mapping change.
export const diffLanguage: monaco.languages.IMonarchLanguage = {
    defaultToken: '',
    tokenPostfix: '.diff',
    tokenizer: {
        root: [
            [FILE_META, 'keyword'],
            [/^(?:---|\+\+\+)(?: .*)?$/, 'keyword'],
            [HUNK_HEADER, { token: 'keyword', next: '@hunk' }],
            [/^\\ No newline at end of file$/, 'comment'],
        ],
        hunk: [
            [HUNK_HEADER, 'keyword'],
            // A new file section returns to the header rules above.
            [FILE_META, { token: 'keyword', next: '@root' }],
            [/^\\ No newline at end of file$/, 'comment'],
            [/^\+.*$/, 'string'],
            [/^-.*$/, 'invalid'],
        ],
    },
};
