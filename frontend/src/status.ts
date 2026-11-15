export function formatCursorPosition(lineNumber: number, column: number): string {
    return `Ln ${lineNumber}, Col ${column}`;
}

export interface LaunchNotice {
    folder: string;
    missing: string[];
}

// noticeFor summarizes what a launch request could not open. A folder alone is
// not an error: it becomes the working folder, shown in the title bar, and the
// file tree that browses it arrives in a later release.
export function noticeFor({ folder, missing }: LaunchNotice): string {
    const parts: string[] = [];

    if (missing.length === 1) {
        parts.push(`Cannot open ${missing[0]}`);
    } else if (missing.length > 1) {
        parts.push(`Cannot open ${missing.length} items`);
    }

    if (folder !== '') {
        parts.push('Folder browsing is not available yet');
    }

    return parts.join(' — ');
}
