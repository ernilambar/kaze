<script lang="ts">
    import EditorPane from './EditorPane.svelte';
    import StatusBar from './StatusBar.svelte';
    import TitleBar from './TitleBar.svelte';

    const APP_NAME = 'Kova';
    const LANGUAGE = 'Plain Text';

    let fileName = $state('untitled.txt');
    let line = $state(1);
    let column = $state(1);
</script>

<div class="app">
    <TitleBar appName={APP_NAME} {fileName} />
    <EditorPane
        onCursorMove={(nextLine, nextColumn) => {
            line = nextLine;
            column = nextColumn;
        }}
    />
    <StatusBar {line} {column} language={LANGUAGE} />
</div>

<style>
    .app {
        display: flex;
        flex-direction: column;
        height: 100vh;
    }
</style>
