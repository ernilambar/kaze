import { ListFiles } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { folderName, isWithin } from './sidebar';

// QuickOpenFile is one row in the palette: the file's absolute path, its name
// and the dimmed location shown beside it.
export interface QuickOpenFile {
    path: string;
    name: string;
    detail: string;
}

// QUICK_OPEN_LIMIT caps the rendered list. Typing narrows it long before a
// virtualized list would be necessary.
export const QUICK_OPEN_LIMIT = 50;

// listFiles asks Go for every regular file under the sidebar roots.
export async function listFiles(): Promise<string[]> {
    return await ListFiles();
}

// listenForQuickOpen runs handler when the File menu asks for the palette.
// Returns an unsubscribe function.
export function listenForQuickOpen(handler: () => void): () => void {
    return EventsOn('kova:quick-open', () => handler());
}

// toQuickOpenFiles turns absolute paths into palette rows. The detail keeps the
// root's name, so files with the same name in different roots stay distinct.
export function toQuickOpenFiles(
    paths: readonly string[],
    roots: readonly string[],
): QuickOpenFile[] {
    return paths.map((path) => toQuickOpenFile(path, roots));
}

export function toQuickOpenFile(path: string, roots: readonly string[]): QuickOpenFile {
    const normalizedPath = stripTrailingSeparators(path.replace(/\\/g, '/'));
    const name = normalizedPath.slice(normalizedPath.lastIndexOf('/') + 1);
    const root = innermostRoot(path, roots);
    if (root === '') {
        return { path, name, detail: parentOf(path.replace(/\\/g, '/')) };
    }

    const relative = path.slice(root.length).replace(/^[\\/]+/, '').replace(/\\/g, '/');
    const parent = parentOf(relative);
    const label = folderName(root);
    return { path, name, detail: parent === '' ? label : `${label}/${parent}` };
}

// fuzzyScore ranks text against query as an in-order subsequence. Boundary and
// consecutive matches win, long candidates lose, and undefined means no match.
export function fuzzyScore(query: string, text: string): number | undefined {
    if (query === '') {
        return 0;
    }
    if (text === '') {
        return undefined;
    }

    const first = query[0].toLowerCase();
    let best: number | undefined;
    // Starting from every possible first match avoids locking onto an early,
    // poorly aligned occurrence when a word boundary appears later.
    for (let index = 0; index < text.length; index++) {
        if (text[index].toLowerCase() !== first) {
            continue;
        }
        const score = greedyScore(query, text, index);
        if (score !== undefined && (best === undefined || score > best)) {
            best = score;
        }
    }
    return best;
}

function greedyScore(query: string, text: string, start: number): number | undefined {
    let score = 0;
    let previous = -1;
    let queryIndex = 0;

    for (let index = start; index < text.length && queryIndex < query.length; index++) {
        const actual = text[index];
        const expected = query[queryIndex];
        if (actual.toLowerCase() !== expected.toLowerCase()) {
            continue;
        }
        if (previous >= 0) {
            score -= (index - previous - 1) * 0.5;
        }
        if (index === previous + 1) {
            score += 6;
        }
        if (isBoundary(text, index)) {
            score += 8;
        }
        if (actual === expected) {
            score += 1;
        }
        previous = index;
        queryIndex++;
    }

    if (queryIndex < query.length) {
        return undefined;
    }
    // A short candidate is more likely the intended file than a long path that
    // happens to contain the same letters.
    return score - text.length * 0.05;
}

function isBoundary(text: string, index: number): boolean {
    if (index === 0) {
        return true;
    }
    const previous = text[index - 1];
    if (/[/\\_.\- ]/.test(previous)) {
        return true;
    }
    const current = text[index];
    return current !== current.toLowerCase() && previous === previous.toLowerCase();
}

// rankQuickOpen matches query against each file's name and its path. Names rank
// above paths, exact names above prefixes, and prefixes above loose matches.
export function rankQuickOpen(
    query: string,
    files: readonly QuickOpenFile[],
    limit: number = QUICK_OPEN_LIMIT,
): QuickOpenFile[] {
    const needle = query.trim();
    if (needle === '') {
        return files.slice(0, limit);
    }

    const lowerNeedle = needle.toLowerCase();
    const matches: Array<{ file: QuickOpenFile; score: number }> = [];

    for (const file of files) {
        const nameScore = fuzzyScore(needle, file.name);
        const pathScore = fuzzyScore(needle, `${file.detail}/${file.name}`);
        if (nameScore === undefined && pathScore === undefined) {
            continue;
        }

        let score = nameScore === undefined ? 0 : nameScore + 20;
        if (pathScore !== undefined) {
            score = Math.max(score, pathScore);
        }

        const lowerName = file.name.toLowerCase();
        if (lowerName === lowerNeedle) {
            score += 100;
        } else if (lowerName.startsWith(lowerNeedle)) {
            score += 40;
        } else if (lowerName.includes(lowerNeedle)) {
            score += 20;
        } else if (file.detail.toLowerCase().includes(lowerNeedle)) {
            score += 10;
        }

        matches.push({ file, score });
    }

    matches.sort((left, right) => {
        if (left.score !== right.score) {
            return right.score - left.score;
        }
        if (left.file.detail.length !== right.file.detail.length) {
            return left.file.detail.length - right.file.detail.length;
        }
        const byDetail = left.file.detail.localeCompare(right.file.detail);
        return byDetail !== 0 ? byDetail : left.file.name.localeCompare(right.file.name);
    });

    return matches.slice(0, limit).map((match) => match.file);
}

function innermostRoot(path: string, roots: readonly string[]): string {
    let innermost = '';
    for (const candidate of roots) {
        const root = stripTrailingSeparators(candidate);
        if (root.length > innermost.length && isWithin(root, path)) {
            innermost = root;
        }
    }
    return innermost;
}

function stripTrailingSeparators(path: string): string {
    return path.length > 1 ? path.replace(/[\\/]+$/, '') : path;
}

function parentOf(path: string): string {
    const index = path.lastIndexOf('/');
    return index <= 0 ? '' : path.slice(0, index);
}
