import './style.css';
import * as monaco from 'monaco-editor';
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
