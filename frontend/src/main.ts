import './style.css';
import { mount } from 'svelte';
import App from './App.svelte';

const target = document.getElementById('app');
if (target === null) {
    throw new Error('Missing #app mount point in index.html.');
}

mount(App, { target });
