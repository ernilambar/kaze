import './style.css';
import * as monaco from 'monaco-editor';
import EditorWorker from 'monaco-editor/editor/editor.worker?worker';
import CssWorker from 'monaco-editor/language/css/css.worker?worker';
import HtmlWorker from 'monaco-editor/language/html/html.worker?worker';
import JsonWorker from 'monaco-editor/language/json/json.worker?worker';
import TsWorker from 'monaco-editor/language/typescript/ts.worker?worker';
import { mount } from 'svelte';
import App from './App.svelte';
import {
    applyStartupTheme,
    readStartupTheme,
    startupThemeData,
    STARTUP_THEME_ID,
    type StartupTheme,
} from './theme';

// DEFAULT_STARTUP_THEME is the built-in dark palette, which is also the
// window's BackgroundColour in windows.go. Settings and theme files are read
// asynchronously after the window is on screen, so the first frame is painted
// from the cached palette (or this one on a fresh install).
const DEFAULT_STARTUP_THEME: StartupTheme = {
    appearance: 'dark',
    background: '#1e1e1e',
    editorBackground: '#181818',
    foreground: '#cccccc',
};

// Vite cannot bundle Monaco's own editor-worker URL: the build inlines its
// source as a data: URL whose relative imports never resolve, so the worker
// behind link detection and word completions never starts. Once getWorker is
// set Monaco asks it for every worker, so the language workers are listed too.
self.MonacoEnvironment = {
    getWorker(_workerId: string, label: string): Worker {
        switch (label) {
            case 'css':
            case 'scss':
            case 'less':
                return new CssWorker();
            case 'html':
            case 'handlebars':
            case 'razor':
                return new HtmlWorker();
            case 'json':
                return new JsonWorker();
            case 'typescript':
            case 'javascript':
                return new TsWorker();
            default:
                return new EditorWorker();
        }
    },
};

const startupTheme = readStartupTheme() ?? DEFAULT_STARTUP_THEME;
applyStartupTheme(startupTheme);
// Monaco defaults to its light `vs` theme, so the editor would paint white until
// the saved palette is read. Selecting the cached one here means the first
// editor is already themed.
monaco.editor.defineTheme(STARTUP_THEME_ID, startupThemeData(startupTheme));
monaco.editor.setTheme(STARTUP_THEME_ID);

const target = document.getElementById('app');
if (target === null) {
    throw new Error('Missing #app mount point in index.html.');
}

mount(App, { target });
