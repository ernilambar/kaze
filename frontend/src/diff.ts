// changedLines compares a buffer with the committed text it was opened from
// and reports which lines in the buffer differ. It is a line-level Myers diff,
// which is what an editor gutter needs: added lines are green, replaced lines
// are marked as modified, and deleted lines have no line of their own to mark.

export type ChangedLineKind = 'added' | 'modified';

export interface ChangedLine {
    // line is 1-based, matching Monaco's line numbers.
    line: number;
    kind: ChangedLineKind;
}

interface Match {
    before: number;
    after: number;
}

// MAX_EDIT_DISTANCE caps the Myers search. A normal edit is a handful of lines
// away from its baseline; a wholesale rewrite past this limit is summarized as
// one changed block instead of spending seconds finding the minimal diff.
const MAX_EDIT_DISTANCE = 400;

// matchLines returns the line pairs the two texts have in common, in order.
// An empty result means the files share no line, including when the search hit
// its edit-distance cap.
function matchLines(before: readonly string[], after: readonly string[]): Match[] {
    const n = before.length;
    const m = after.length;
    const limit = Math.min(n + m, MAX_EDIT_DISTANCE);

    // v maps a diagonal k = x - y to the furthest x reached on it.
    const v = new Map<number, number>();
    v.set(1, 0);
    const trace: Map<number, number>[] = [];
    let found = -1;

    for (let distance = 0; distance <= limit && found < 0; distance++) {
        trace.push(new Map(v));
        for (let k = -distance; k <= distance; k += 2) {
            const down = v.get(k + 1) ?? Number.NEGATIVE_INFINITY;
            const right = v.get(k - 1) ?? Number.NEGATIVE_INFINITY;
            let x: number;
            if (k === -distance || (k !== distance && right < down)) {
                x = down === Number.NEGATIVE_INFINITY ? 0 : down;
            } else {
                x = right === Number.NEGATIVE_INFINITY ? 0 : right + 1;
            }
            let y = x - k;
            while (x < n && y < m && before[x] === after[y]) {
                x++;
                y++;
            }
            v.set(k, x);
            if (x >= n && y >= m) {
                found = distance;
                break;
            }
        }
    }

    if (found < 0) {
        return [];
    }

    const matches: Match[] = [];
    let x = n;
    let y = m;
    for (let distance = found; distance >= 0; distance--) {
        const previous = trace[distance];
        const k = x - y;
        const down = previous.get(k + 1) ?? Number.NEGATIVE_INFINITY;
        const right = previous.get(k - 1) ?? Number.NEGATIVE_INFINITY;
        const previousK =
            k === -distance || (k !== distance && right < down) ? k + 1 : k - 1;
        const previousX = previous.get(previousK) ?? 0;
        const previousY = previousX - previousK;
        while (x > previousX && y > previousY) {
            x--;
            y--;
            matches.push({ before: x, after: y });
        }
        x = previousX;
        y = previousY;
    }
    matches.reverse();
    return matches;
}

// fallbackChanged marks the middle of the buffer as one changed block. It runs
// only when the minimal diff would be larger than the cap, where the exact
// boundaries are not worth a long computation.
function fallbackChanged(before: readonly string[], after: readonly string[]): ChangedLine[] {
    let start = 0;
    while (start < before.length && start < after.length && before[start] === after[start]) {
        start++;
    }
    let beforeEnd = before.length;
    let afterEnd = after.length;
    while (beforeEnd > start && afterEnd > start && before[beforeEnd - 1] === after[afterEnd - 1]) {
        beforeEnd--;
        afterEnd--;
    }

    const kind: ChangedLineKind = start < beforeEnd ? 'modified' : 'added';
    const changed: ChangedLine[] = [];
    for (let line = start; line < afterEnd; line++) {
        changed.push({ line: line + 1, kind });
    }
    return changed;
}

// changedLines is the public entry point: the 1-based lines of after that were
// added or replaced, and how each one changed.
export function changedLines(before: string, after: string): ChangedLine[] {
    if (before === after) {
        return [];
    }

    // Normalizing CRLF keeps a file stored with CRLF line endings from
    // comparing every line against a blob that git stores with LF.
    const beforeLines = before.replace(/\r\n/g, '\n').split('\n');
    const afterLines = after.replace(/\r\n/g, '\n').split('\n');
    const matches = matchLines(beforeLines, afterLines);
    if (matches.length === 0) {
        return fallbackChanged(beforeLines, afterLines);
    }

    const changed: ChangedLine[] = [];
    let beforeIndex = 0;
    let afterIndex = 0;

    const addGap = (beforeCount: number, afterCount: number): void => {
        if (afterCount === 0) {
            return;
        }
        const kind: ChangedLineKind = beforeCount > 0 ? 'modified' : 'added';
        for (let line = afterIndex; line < afterIndex + afterCount; line++) {
            changed.push({ line: line + 1, kind });
        }
    };

    for (const match of matches) {
        const beforeCount = match.before - beforeIndex;
        const afterCount = match.after - afterIndex;
        addGap(beforeCount, afterCount);
        beforeIndex = match.before + 1;
        afterIndex = match.after + 1;
    }
    addGap(beforeLines.length - beforeIndex, afterLines.length - afterIndex);

    return changed;
}
