export function formatCursorPosition(
    lineNumber: number,
    column: number,
    selectionLength: number = 0,
): string {
    const position = `Ln ${lineNumber}, Col ${column}`;
    return selectionLength > 0 ? `${position} (${selectionLength} selected)` : position;
}

// lineEndingLabel is the status bar's name for a document's line ending. The
// control that prints it switches the buffer between the two.
export function lineEndingLabel(eol: string): string {
    return eol === 'crlf' ? 'CRLF' : 'LF';
}

export interface LaunchNotice {
    missing: string[];
}

// noticeFor summarizes what a launch request could not open. Folders are not
// errors: each one becomes a root in the sidebar, so only missing paths are
// reported.
export function noticeFor({ missing }: LaunchNotice): string {
    if (missing.length === 1) {
        return `Cannot open ${missing[0]}`;
    }
    if (missing.length > 1) {
        return `Cannot open ${missing.length} items`;
    }
    return '';
}
