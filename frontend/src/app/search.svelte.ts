import { tick } from 'svelte';
import { ReplaceInProject, SearchProject } from '../../bindings/kaze/internal/app/app';
import type { SearchMatch } from '../../bindings/kaze/internal/find/models';
import type { OpenDocument } from '../document';
import {
    expectedModified,
    groupSearchMatches,
    replaceConfirmMessage,
    replaceNotice,
    replaceText,
    searchExpression,
    searchSummary,
    stepSearchResult,
} from '../search';
import { errorMessage } from '../sidebar';
import { findDocumentByPath, updateDocument } from '../tabs';
import type { WindowContext } from './context.svelte';
import type { GitController } from './git.svelte';
import type { Prompts } from './prompts.svelte';

// SearchHost is what Find in Files needs from the window that is not search
// state: opening a file, and rereading the tree after a replace.
export interface SearchHost {
    openFile: (path: string) => Promise<string>;
    refreshTree: () => void;
}

// SearchController is Find in Files state. The results tab is open while
// results exist, even when another tab is active.
export class SearchController {
    private readonly ctx: WindowContext;
    private readonly prompts: Prompts;
    private readonly git: GitController;
    private readonly host: SearchHost;

    // Find in Files state. The results tab is open while results exist, even
    // when another tab is active.
    open = $state(false);
    query = $state('');
    include = $state('');
    excludeGlobs = $state('');
    replacement = $state('');
    regex = $state(false);
    caseSensitive = $state(false);
    matches = $state<SearchMatch[]>([]);
    truncated = $state(false);

    // resultIndex is the position F4 last opened in matches, or -1
    // before the first step. A new search resets it.
    resultIndex = $state(-1);

    // searchedParams records the query that produced matches. Replace
    // All uses these rather than the live fields, so changing the query without
    // searching again cannot replace text the results do not describe.
    searchedParams = $state({
        query: '',
        regex: false,
        caseSensitive: false,
        include: '',
        exclude: '',
    });

    busy = $state(false);
    error = $state('');
    focusToken = $state(0);

    groups = $derived(groupSearchMatches(this.matches));

    label = $derived(
        searchSummary(this.groups.length, this.matches.length, this.truncated),
    );

    closeSearchTab = (): void => {
        this.open = false;
        this.ctx.searchActive = false;
        this.matches = [];
        this.resultIndex = -1;
        this.error = '';
    };

    runSearch = async (): Promise<void> => {
        const query = this.query.trim();
        if (query === '') {
            this.matches = [];
            this.resultIndex = -1;
            this.error = '';
            this.searchedParams = {
                query: '',
                regex: false,
                caseSensitive: false,
                include: '',
                exclude: '',
            };
            return;
        }
        this.busy = true;
        this.error = '';
        try {
            const result = await SearchProject(this.ctx.windowName, {
                query,
                regex: this.regex,
                case_sensitive: this.caseSensitive,
                include: this.include,
                exclude: this.excludeGlobs,
                replacement: '',
                expected_modified: null,
                open_paths: null,
            });
            this.matches = result.matches ?? [];
            this.resultIndex = -1;
            this.truncated = result.truncated;
            this.searchedParams = {
                query,
                regex: this.regex,
                caseSensitive: this.caseSensitive,
                include: this.include,
                exclude: this.excludeGlobs,
            };
        } catch (error) {
            this.matches = [];
            this.resultIndex = -1;
            this.error = errorMessage(error);
        } finally {
            this.busy = false;
        }
    };

    // replaceInFiles is the safe Replace All. It confirms the scope first, then
    // lets Go write the files whose mtime still matches the search; files that
    // changed since are skipped and named in the result. Open buffers get the
    // same edit through Monaco so it stays unsaved and undoable, and the app
    // never reloads a buffer out from under the user's undo stack.
    replaceInFiles = async (): Promise<void> => {
        const query = this.searchedParams.query;
        if (query === '' || this.matches.length === 0) {
            return;
        }
        const files = this.groups.length;
        const matches = this.matches.length;
        const confirmed = await this.prompts.askConfirm(
            'Replace All in Files',
            replaceConfirmMessage(matches, files, this.truncated),
            'Replace All',
        );
        if (!confirmed) {
            return;
        }

        this.busy = true;
        this.error = '';
        try {
            const expected = expectedModified(this.matches);
            const expression = searchExpression(
                query,
                this.searchedParams.regex,
                this.searchedParams.caseSensitive,
            );
            // Files open in this window are edited in their Monaco buffers,
            // so tell Go to leave them on disk. They stay unsaved and undoable.
            // openPaths uses the search's spelling so it matches the path Go
            // walks, even when the tab was opened through a symlink.
            const openPaths: string[] = [];
            const openTabs: OpenDocument[] = [];
            const seenTabs = new Set<string>();
            for (const group of this.groups) {
                const tab = findDocumentByPath(this.ctx.tabs, group.path);
                if (tab !== undefined && tab.kind === 'text' && tab.path !== '') {
                    openPaths.push(group.path);
                    // Two roots can name the same file, and both would map to
                    // one tab; edit its buffer once.
                    if (!seenTabs.has(tab.id)) {
                        seenTabs.add(tab.id);
                        openTabs.push(tab);
                    }
                }
            }
            const result = await ReplaceInProject(this.ctx.windowName, {
                query,
                regex: this.searchedParams.regex,
                case_sensitive: this.searchedParams.caseSensitive,
                include: this.searchedParams.include,
                exclude: this.searchedParams.exclude,
                replacement: this.replacement,
                expected_modified: expected,
                open_paths: openPaths,
            });

            // Apply the same edit to each open buffer. A dirty buffer's model
            // may hold unsaved text the search never saw, so replace in the
            // buffer's current text; Monaco keeps the undo history and marks
            // the tab dirty through onDirtyChange.
            for (const tab of openTabs) {
                const current = this.ctx.editorPane?.value(tab.id) ?? tab.content;
                const replaced = replaceText(
                    current,
                    expression,
                    this.replacement,
                    this.searchedParams.regex,
                );
                if (replaced === current) {
                    continue;
                }
                const updated = this.ctx.editorPane?.replaceInBuffer(tab.id, replaced);
                if (updated === undefined) {
                    // A tab restored from the session may never have been
                    // displayed, so it has no Monaco model yet. Store the new
                    // text on the tab so the model is created already dirty.
                    this.ctx.tabs = updateDocument(this.ctx.tabs, tab.id, {
                        content: replaced,
                        savedContent: tab.savedContent ?? tab.content,
                        dirty: true,
                    });
                }
            }

            this.host.refreshTree();
            await this.git.refreshGitStatus();
            await this.git.refreshDiffTabs();
            this.ctx.notice = replaceNotice(result.matches, result.files, result.skipped ?? [], result.failed ?? []);
            await this.runSearch();
        } catch (error) {
            this.error = errorMessage(error);
        } finally {
            this.busy = false;
        }
    };

    // openSearchMatch switches to the file, then puts the caret on the hit.
    // The model is created by the tab effect, so the reveal waits a tick.
    openSearchMatch = async (path: string, line: number, column: number): Promise<void> => {
        const id = await this.host.openFile(path);
        if (id === '') {
            return;
        }
        this.ctx.searchActive = false;
        await tick();
        this.ctx.editorPane?.revealLine(id, line, column);
    };

    // stepFindResult opens the next or previous Find in Files hit. The index
    // advances before the file is opened, so holding F4 cannot open the same
    // hit twice while the first read is still in flight.
    stepFindResult = async (direction: 1 | -1): Promise<void> => {
        if (this.matches.length === 0) {
            this.ctx.notice = 'No search results to step through.';
            return;
        }
        const index = stepSearchResult(this.resultIndex, this.matches.length, direction);
        this.resultIndex = index;
        const match = this.matches[index];
        if (match === undefined) {
            return;
        }
        await this.openSearchMatch(match.path, match.line, match.column);
    };

    constructor(ctx: WindowContext, prompts: Prompts, git: GitController, host: SearchHost) {
        this.ctx = ctx;
        this.prompts = prompts;
        this.git = git;
        this.host = host;
    }
}
