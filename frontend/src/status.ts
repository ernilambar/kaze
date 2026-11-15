export function formatCursorPosition(lineNumber: number, column: number): string {
    return `Ln ${lineNumber}, Col ${column}`;
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
