import type * as monaco from 'monaco-editor';

// LOG_LANGUAGE_ID is the id Monaco registers, and the one the status bar and
// Change Language Mode list read back from the registry.
export const LOG_LANGUAGE_ID = 'log';

// LOG_EXTENSIONS is what Auto Detect matches. Everything else (the palette
// entry, the status label) follows from the registration in editor.ts.
export const LOG_EXTENSIONS = ['.log'];

// LOG_LANGUAGE is a generic log tokenizer: it colors the structured parts most
// log lines share and leaves prose on the default foreground. The token names
// are the ones syntaxRules in theme.ts styles, so every theme colors it without
// a mapping change.
export const logLanguage: monaco.languages.IMonarchLanguage = {
    ignoreCase: true,
    tokenizer: {
        root: [
            // Timestamps: ISO 8601, bracketed dates, syslog months and bare
            // times. The more specific forms must win over the generic number
            // rule below.
            [
                /\d{4}-\d{2}-\d{2}[t ]\d{2}:\d{2}:\d{2}(?:[.,]\d{1,9})?(?:z|[+-]\d{2}:?\d{2})?/,
                'number',
            ],
            [
                /\[\s*\d{4}-\d{2}-\d{2}[t ]\d{2}:\d{2}:\d{2}(?:[.,]\d{1,9})?(?:z|[+-]\d{2}:?\d{2})?\s*\]/,
                'number',
            ],
            [/\d{4}-\d{2}-\d{2}\b/, 'number'],
            [
                /\b(?:jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec) +\d{1,2} +\d{1,2}:\d{2}:\d{2}\b/,
                'number',
            ],
            [/\b\d{1,2}:\d{2}(?::\d{2})?(?:[.,]\d{1,9})?\b/, 'number'],
            [/\[\s*\d{1,2}:\d{2}(?::\d{2})?(?:[.,]\d{1,9})?\s*\]/, 'number'],

            // A bracketed level (nginx and uvicorn write [error]) gets the same
            // color as a bare one, so these precede the generic bracket rule.
            [/\[\s*(?:alert|crit|critical|emerg|emergency|err|error|fatal|severe)\s*\]/, 'keyword'],
            [/\[\s*(?:warn|warning|notice)\s*\]/, 'type'],
            [/\[\s*(?:info|information|note)\s*\]/, 'constant'],
            [/\[\s*(?:debug|trace|verbose|silly)\s*\]/, 'comment'],
            // Thread names, pids and other bracketed headers.
            [/\[[a-z_][\w .:@/-]{0,48}\]/, 'type'],

            // Levels without brackets, as in "... - INFO -" or "ERROR: ...".
            [/\b(?:alert|crit|critical|emerg|emergency|err|error|fatal|severe)\b/, 'keyword'],
            [/\b(?:warn|warning|notice)\b/, 'type'],
            [/\b(?:info|information|note)\b/, 'constant'],
            [/\b(?:debug|trace|verbose|silly)\b/, 'comment'],

            // Structured fields: logfmt keys, JSON keys and quoted values.
            [/[\w.-]+(?=\s*=)/, 'key'],
            [/"([^"\\]|\\.)*"(?=\s*:)/, 'key'],
            [/"([^"\\]|\\.)*"/, 'string'],
            [/'([^'\\]|\\.)*'/, 'string'],

            [/\b[a-z][a-z0-9+.-]*:\/\/[^\s"'<>]+/, 'string'],
            [/\b\d{1,3}(?:\.\d{1,3}){3}(?::\d{1,5})?\b/, 'variable'],
            [
                /\b(?:0x[0-9a-f]+|\d+(?:\.\d+)?(?:e[+-]?\d+)?)(?:ns|us|µs|ms|s|m|h|kb|mb|gb|tb)?\b/,
                'number',
            ],
            // Dotted names: loggers, class names and host names.
            [/[a-z_$][\w$]*(?:\.[a-z_$][\w$]*)+/, 'type'],
            [/(?:[a-z]:)?(?:[\\/][\w.@+-]+){2,}/, 'variable'],

            [/[{}()\[\]<>]/, 'delimiter.bracket'],
            [/[=:,;]/, 'delimiter'],
        ],
    },
};
