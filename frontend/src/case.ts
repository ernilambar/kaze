// CaseTransform names an Edit menu transform. The app:transform-case event
// carries this id.
export type CaseTransform =
    | 'upper'
    | 'lower'
    | 'title'
    | 'camel'
    | 'pascal'
    | 'snake'
    | 'kebab'
    | 'constant';

const WORD = /[A-Z]+(?![a-z])|[A-Z]?[a-z]+|\d+/g;

// CASE_TRANSFORMS is the Edit menu's Transform Case order, shared by the
// command palette and the menu event handler.
export const CASE_TRANSFORMS: readonly { id: CaseTransform; label: string }[] = [
    { id: 'upper', label: 'Uppercase' },
    { id: 'lower', label: 'Lowercase' },
    { id: 'title', label: 'Title Case' },
    { id: 'camel', label: 'camelCase' },
    { id: 'pascal', label: 'PascalCase' },
    { id: 'snake', label: 'snake_case' },
    { id: 'kebab', label: 'kebab-case' },
    { id: 'constant', label: 'CONSTANT_CASE' },
];

// splitWords breaks an identifier or a prose phrase into words. A run of
// capitals stays together (HTTPServer → HTTP, Server) so a constant can be
// converted back to camelCase without losing its boundaries.
export function splitWords(text: string): string[] {
    return text.match(WORD) ?? [];
}

function capitalize(word: string): string {
    const lower = word.toLowerCase();
    return lower.charAt(0).toUpperCase() + lower.slice(1);
}

// transformCase converts text to one of the shapes the Edit menu offers. It is
// applied to the selection, or to the whole buffer when nothing is selected.
export function transformCase(text: string, transform: CaseTransform): string {
    switch (transform) {
        case 'upper':
            return text.toUpperCase();
        case 'lower':
            return text.toLowerCase();
        case 'title':
            // Only the first letter of each whitespace-delimited word moves, so
            // punctuation and inner capitals ("iPhone") stay as written.
            return text.replace(
                /\S+/g,
                (word) => word.charAt(0).toUpperCase() + word.slice(1),
            );
        case 'camel': {
            const words = splitWords(text);
            return words.length === 0
                ? ''
                : words[0].toLowerCase() + words.slice(1).map(capitalize).join('');
        }
        case 'pascal':
            return splitWords(text).map(capitalize).join('');
        case 'snake':
            return splitWords(text)
                .map((word) => word.toLowerCase())
                .join('_');
        case 'kebab':
            return splitWords(text)
                .map((word) => word.toLowerCase())
                .join('-');
        case 'constant':
            return splitWords(text)
                .map((word) => word.toUpperCase())
                .join('_');
    }
}
