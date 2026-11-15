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
