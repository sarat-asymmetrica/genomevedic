/**
 * GenomeVedic.ai - Main Entry Point
 *
 * Initializes the Svelte app with dark theme and WebGL renderer
 */

import App from './App.svelte';

const app = new App({
  target: document.getElementById('app'),
  props: {
    appName: 'GenomeVedic.ai',
    version: '1.0.0'
  }
});

export default app;
