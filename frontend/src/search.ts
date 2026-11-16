import type { SearchMatch } from '../bindings/kova/models';

// SearchFileGroup is the results list's unit: one file and every match in it.
// Grouping happens here so the panel and its tests share one shape.
export interface SearchFileGroup {
    path: string;
    name: string;
    matches: SearchMatch[];
}

// groupSearchMatches collects matches into files, preserving the order the
// backend reported them in (paths sorted, lines in file order).
export function groupSearchMatches(matches: readonly SearchMatch[]): SearchFileGroup[] {
    const groups: SearchFileGroup[] = [];
    const index = new Map<string, SearchFileGroup>();
    for (const match of matches) {
        let group = index.get(match.path);
        if (group === undefined) {
            group = { path: match.path, name: match.name, matches: [] };
            index.set(match.path, group);
            groups.push(group);
        }
        group.matches.push(match);
    }
    return groups;
}

// stepSearchResult returns the index of the next result to open. A negative
// current means nothing has been visited yet: F4 then starts at the first
// result and Shift+F4 at the last one. Both ends wrap.
export function stepSearchResult(current: number, count: number, direction: 1 | -1): number {
    if (count <= 0) {
        return -1;
    }
    if (current < 0 || current >= count) {
        return direction > 0 ? 0 : count - 1;
    }
    return (current + direction + count) % count;
}

// searchSummary is the line above the results. It reports matched files rather
// than searched files, because that is what the user can act on.
export function searchSummary(groups: number, matches: number, truncated: boolean): string {
    if (matches === 0) {
        return 'No results.';
    }
    const files = `${groups} ${groups === 1 ? 'file' : 'files'}`;
    const hits = `${matches} ${matches === 1 ? 'match' : 'matches'}`;
    return truncated ? `${hits} in ${files} (limited)` : `${hits} in ${files}`;
}

// expectedModified maps each matched file to the mtime the search read it at.
// Replace All hands this to Go so it can refuse to write a file that changed
// after the results were produced.
export function expectedModified(matches: readonly SearchMatch[]): Record<string, number> {
    const map: Record<string, number> = {};
    for (const match of matches) {
        map[match.path] = match.modified;
    }
    return map;
}

// REGEX_SPECIAL is the set of characters a literal query must escape so the
// frontend's buffer edit matches the backend's regexp.QuoteMeta behaviour.
const REGEX_SPECIAL = /[.*+?^${}()|[\]\\]/g;

// searchExpression builds the regular expression for one query: a literal
// query is quoted so a period is a period, and the case option becomes a flag.
// This mirrors searchExpression in search.go.
export function searchExpression(query: string, regex: boolean, caseSensitive: boolean): RegExp {
    const source = regex ? query : query.replace(REGEX_SPECIAL, '\\$&');
    return new RegExp(source, caseSensitive ? 'g' : 'gi');
}

// escapeReplacement escapes a literal replacement so $ sequences stay text,
// matching ReplaceAllLiteral in the backend. In regex mode the replacement is
// used as-is, so $1 can insert a capture group.
export function replaceText(
    text: string,
    expression: RegExp,
    replacement: string,
    regex: boolean,
): string {
    return regex ? text.replace(expression, replacement) : text.replace(expression, () => replacement);
}

// replaceConfirmMessage reports exactly what Replace All will touch, so the
// user can confirm with the match and file counts in front of them.
export function replaceConfirmMessage(matches: number, files: number, truncated: boolean): string {
    const hits = `${matches} ${matches === 1 ? 'match' : 'matches'}`;
    const targets = `${files} ${files === 1 ? 'file' : 'files'}`;
    const limit = truncated ? ' Results were limited, so the counts shown are a minimum.' : '';
    return `Replace ${hits} in ${targets}? Open files become unsaved edits you can undo.${limit}`;
}

// replaceNotice summarizes what Replace All did. Skipped files changed on disk
// after the search and were left untouched, so they are named in the notice.
export function replaceNotice(replaced: number, files: number, skipped: readonly string[]): string {
    let text = `Replaced ${replaced} in ${files} ${files === 1 ? 'file' : 'files'}.`;
    if (skipped.length > 0) {
        text += ` Skipped ${skipped.length} ${skipped.length === 1 ? 'file' : 'files'} changed since the search: ${skipped.join(', ')}.`;
    }
    return text;
}
