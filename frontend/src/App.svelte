<script>
  /**
   * GenomeVedic.ai - Main App Component
   *
   * Features:
   * - Dark theme UI
   * - WebGL particle renderer integration
   * - Side panel for controls and upload
   * - Stats panel for performance metrics
   */

  import { onMount } from 'svelte';
  import ParticleRenderer from './renderer/particle_renderer.js';
  import QuaternionCamera from './camera/quaternion_camera.js';
  import MouseControls from './camera/mouse_controls.js';
  import KeyboardControls from './camera/keyboard_controls.js';
  import FASTQUpload from './components/FASTQUpload.svelte';
  import VisualizationControls from './components/VisualizationControls.svelte';
  import StatsPanel from './components/StatsPanel.svelte';

  // Props
  export let appName = 'GenomeVedic.ai';
  export let version = '1.0.0';

  // State
  let canvas;
  let renderer;
  let camera;
  let mouseControls;
  let keyboardControls;
  let animationFrameId;

  let showUploadPanel = true;
  let showControlsPanel = true;
  let showStatsPanel = true;

  let stats = {
    fps: 0,
    frameTime: 0,
    particleCount: 0,
    visibleParticles: 0,
    memoryUsage: 0
  };

  // Performance monitoring
  let lastFrameTime = performance.now();
  let frameCount = 0;
  let fpsUpdateInterval = 0;

  // Initialize WebGL renderer
  onMount(() => {
    // Create WebGL context
    const gl = canvas.getContext('webgl2', {
      antialias: true,
      alpha: false,
      depth: true,
      stencil: false,
      preserveDrawingBuffer: false,
      powerPreference: 'high-performance'
    });

    if (!gl) {
      alert('WebGL 2.0 not supported. Please use a modern browser.');
      return;
    }

    // Initialize renderer
    renderer = new ParticleRenderer(gl);

    // Initialize camera
    camera = new QuaternionCamera();
    camera.setPosition([0, 0, 1000]);
    camera.lookAt([0, 0, 0]);

    // Initialize controls
    mouseControls = new MouseControls(canvas, camera);
    keyboardControls = new KeyboardControls(camera);

    // Generate sample particles (golden spiral)
    generateSampleParticles();

    // Start render loop
    startRenderLoop();

    // Cleanup on unmount
    return () => {
      if (animationFrameId) {
        cancelAnimationFrame(animationFrameId);
      }
      mouseControls.dispose();
      keyboardControls.dispose();
    };
  });

  function generateSampleParticles() {
    const particleCount = 50000;
    const particles = [];

    const goldenAngle = 137.5 * Math.PI / 180;

    for (let i = 0; i < particleCount; i++) {
      const t = i / particleCount;
      const radius = 800 * Math.sqrt(t);
      const angle = i * goldenAngle;
      const height = (t - 0.5) * 400;

      const x = radius * Math.cos(angle);
      const z = radius * Math.sin(angle);
      const y = height;

      // Color based on position (Vedic digital root)
      const digitalRoot = ((i % 9) + 1);
      const hue = digitalRoot / 9;

      const r = Math.abs(Math.cos(hue * Math.PI * 2));
      const g = Math.abs(Math.cos((hue + 0.33) * Math.PI * 2));
      const b = Math.abs(Math.cos((hue + 0.67) * Math.PI * 2));

      particles.push({
        x, y, z,
        r, g, b,
        size: 3.0 + Math.random() * 2.0
      });
    }

    renderer.updateParticles(particles);
    stats.particleCount = particleCount;
    stats.visibleParticles = particleCount;
  }

  function startRenderLoop() {
    function render(currentTime) {
      // Update delta time
      const deltaTime = (currentTime - lastFrameTime) / 1000;
      lastFrameTime = currentTime;

      // Update camera
      camera.update(deltaTime);

      // Render scene
      renderer.render(camera);

      // Update stats
      frameCount++;
      fpsUpdateInterval += deltaTime;

      if (fpsUpdateInterval >= 0.5) {
        stats.fps = Math.round(frameCount / fpsUpdateInterval);
        stats.frameTime = (fpsUpdateInterval / frameCount) * 1000;
        frameCount = 0;
        fpsUpdateInterval = 0;
      }

      // Continue loop
      animationFrameId = requestAnimationFrame(render);
    }

    animationFrameId = requestAnimationFrame(render);
  }

  function handleFileUploaded(event) {
    const { file, metadata } = event.detail;
    console.log('File uploaded:', file.name, 'Reads:', metadata.readCount);

    // TODO: Parse FASTQ and update particles
    // For now, regenerate sample particles
    generateSampleParticles();
  }

  function handleControlChange(event) {
    const { control, value } = event.detail;

    switch (control) {
      case 'colorMode':
        console.log('Color mode changed:', value);
        // TODO: Update particle colors
        break;
      case 'lodLevel':
        console.log('LOD level changed:', value);
        // TODO: Update LOD
        break;
      case 'cameraSpeed':
        camera.movementSpeed = value;
        break;
      case 'zoomLevel':
        console.log('Zoom level changed:', value);
        // TODO: Update zoom
        break;
    }
  }

  function togglePanel(panel) {
    switch (panel) {
      case 'upload':
        showUploadPanel = !showUploadPanel;
        break;
      case 'controls':
        showControlsPanel = !showControlsPanel;
        break;
      case 'stats':
        showStatsPanel = !showStatsPanel;
        break;
    }
  }
</script>

<main>
  <!-- Header -->
  <header>
    <div class="logo">
      <span class="logo-icon">🧬</span>
      <span class="logo-text">{appName}</span>
      <span class="version">v{version}</span>
    </div>

    <div class="header-controls">
      <button class="icon-btn" on:click={() => togglePanel('upload')} title="Toggle Upload Panel">
        📁
      </button>
      <button class="icon-btn" on:click={() => togglePanel('controls')} title="Toggle Controls Panel">
        🎛️
      </button>
      <button class="icon-btn" on:click={() => togglePanel('stats')} title="Toggle Stats Panel">
        📊
      </button>
    </div>
  </header>

  <!-- Main Canvas -->
  <canvas bind:this={canvas}></canvas>

  <!-- Side Panels -->
  {#if showUploadPanel}
    <div class="panel panel-left">
      <FASTQUpload on:fileUploaded={handleFileUploaded} />
    </div>
  {/if}

  {#if showControlsPanel}
    <div class="panel panel-right">
      <VisualizationControls on:controlChange={handleControlChange} />
    </div>
  {/if}

  {#if showStatsPanel}
    <div class="panel panel-bottom">
      <StatsPanel {stats} />
    </div>
  {/if}
</main>

<style>
  main {
    width: 100%;
    height: 100%;
    position: relative;
    background: #0a0a0a;
  }

  /* Header */
  header {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 60px;
    background: rgba(20, 20, 20, 0.95);
    backdrop-filter: blur(10px);
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 20px;
    z-index: 100;
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 20px;
    font-weight: 600;
  }

  .logo-icon {
    font-size: 28px;
  }

  .logo-text {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
  }

  .version {
    font-size: 12px;
    color: #888;
    font-weight: 400;
  }

  .header-controls {
    display: flex;
    gap: 10px;
  }

  .icon-btn {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: #e0e0e0;
    padding: 8px 12px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 18px;
    transition: all 0.2s;
  }

  .icon-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    border-color: rgba(255, 255, 255, 0.2);
    transform: translateY(-1px);
  }

  .icon-btn:active {
    transform: translateY(0);
  }

  /* Canvas */
  canvas {
    position: absolute;
    top: 60px;
    left: 0;
    width: 100%;
    height: calc(100% - 60px);
    background: #0a0a0a;
  }

  /* Panels */
  .panel {
    position: absolute;
    background: rgba(20, 20, 20, 0.95);
    backdrop-filter: blur(10px);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    padding: 20px;
    z-index: 50;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
  }

  .panel-left {
    top: 80px;
    left: 20px;
    width: 350px;
    max-height: calc(100vh - 200px);
    overflow-y: auto;
  }

  .panel-right {
    top: 80px;
    right: 20px;
    width: 300px;
    max-height: calc(100vh - 200px);
    overflow-y: auto;
  }

  .panel-bottom {
    bottom: 20px;
    left: 50%;
    transform: translateX(-50%);
    width: auto;
    min-width: 400px;
  }

  /* Scrollbar styling */
  .panel::-webkit-scrollbar {
    width: 8px;
  }

  .panel::-webkit-scrollbar-track {
    background: rgba(255, 255, 255, 0.05);
    border-radius: 4px;
  }

  .panel::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.2);
    border-radius: 4px;
  }

  .panel::-webkit-scrollbar-thumb:hover {
    background: rgba(255, 255, 255, 0.3);
  }
</style>
