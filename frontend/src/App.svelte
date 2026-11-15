<script lang="ts">
    import { onMount } from 'svelte';
    import EditorPane from './EditorPane.svelte';
    import StatusBar from './StatusBar.svelte';
    import TitleBar from './TitleBar.svelte';
    import {
        folderLabel,
        listenForLaunch,
        readDocument,
        shownName,
        type LaunchRequest,
        type OpenDocument,
    } from './document';
    import { noticeFor } from './status';

    const APP_NAME = 'Kova';
    const LANGUAGE = 'Plain Text';

    let current = $state<OpenDocument>();
    let folder = $state('');
    let notice = $state('');

    let line = $state(1);
    let column = $state(1);

    let folderName = $derived(folderLabel(folder));

    // One tab needs a tab bar, which lands in the next phase: until then the
    // first readable path is opened and the rest are reported as closed.
    async function applyLaunch(request: LaunchRequest) {
        if (request.folder !== '') {
            folder = request.folder;
        }
        notice = noticeFor(request);

        const path = request.files[0];
        if (path === undefined) {
            return;
        }
        const opened = await readDocument(path);
        if (opened !== undefined) {
            current = opened;
        }
    }

    onMount(() => {
        let unsubscribe: (() => void) | undefined;
        let disposed = false;

        void listenForLaunch((request) => void applyLaunch(request)).then((off) => {
            if (disposed) {
                off();
                return;
            }
            unsubscribe = off;
        });

        return () => {
            disposed = true;
            unsubscribe?.();
        };
    });
</script>

<div class="app">
    <TitleBar appName={APP_NAME} fileName={shownName(current)} folder={folderName} />
    <EditorPane
        content={current?.content}
        onCursorMove={(nextLine, nextColumn) => {
            line = nextLine;
            column = nextColumn;
        }}
    />
    <StatusBar {line} {column} language={LANGUAGE} {notice} folder={folderName} />
</div>

<style>
    .app {
        display: flex;
        flex-direction: column;
        height: 100vh;
    }
</style>
