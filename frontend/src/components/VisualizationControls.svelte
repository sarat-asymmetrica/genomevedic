<script>
  /**
   * Visualization Controls Component
   *
   * Features:
   * - Color mode selection (GC content, quality, mutations, annotations)
   * - LOD (Level of Detail) slider
   * - Camera speed control
   * - Zoom level selector (5 levels: Genome → Chromosome → Gene → Exon → Nucleotide)
   * - Particle density control
   * - Show/hide features toggles
   */

  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  // State
  let colorMode = 'gc-content';
  let lodLevel = 2;
  let cameraSpeed = 100;
  let zoomLevel = 2; // Gene level
  let particleDensity = 50;
  let showMutations = true;
  let showAnnotations = true;
  let showTrails = false;

  // Color modes
  const colorModes = [
    { id: 'gc-content', name: 'GC Content', icon: '🧬' },
    { id: 'quality', name: 'Quality Scores', icon: '📊' },
    { id: 'mutations', name: 'Mutations', icon: '🔴' },
    { id: 'annotations', name: 'Gene Annotations', icon: '🏷️' },
    { id: 'digital-root', name: 'Digital Root', icon: '✨' }
  ];

  // Zoom levels
  const zoomLevels = [
    { id: 0, name: 'Genome', range: '3B bp', icon: '🌍' },
    { id: 1, name: 'Chromosome', range: '250M bp', icon: '🧵' },
    { id: 2, name: 'Gene', range: '100K bp', icon: '🧬' },
    { id: 3, name: 'Exon', range: '1K bp', icon: '📍' },
    { id: 4, name: 'Nucleotide', range: '1-100 bp', icon: '🔬' }
  ];

  // Handlers
  function handleColorModeChange(mode) {
    colorMode = mode;
    dispatch('controlChange', { control: 'colorMode', value: mode });
  }

  function handleLODChange(e) {
    lodLevel = parseInt(e.target.value);
    dispatch('controlChange', { control: 'lodLevel', value: lodLevel });
  }

  function handleCameraSpeedChange(e) {
    cameraSpeed = parseInt(e.target.value);
    dispatch('controlChange', { control: 'cameraSpeed', value: cameraSpeed });
  }

  function handleZoomLevelChange(level) {
    zoomLevel = level;
    dispatch('controlChange', { control: 'zoomLevel', value: level });
  }

  function handleParticleDensityChange(e) {
    particleDensity = parseInt(e.target.value);
    dispatch('controlChange', { control: 'particleDensity', value: particleDensity });
  }

  function handleToggleMutations() {
    showMutations = !showMutations;
    dispatch('controlChange', { control: 'showMutations', value: showMutations });
  }

  function handleToggleAnnotations() {
    showAnnotations = !showAnnotations;
    dispatch('controlChange', { control: 'showAnnotations', value: showAnnotations });
  }

  function handleToggleTrails() {
    showTrails = !showTrails;
    dispatch('controlChange', { control: 'showTrails', value: showTrails });
  }

  function resetDefaults() {
    colorMode = 'gc-content';
    lodLevel = 2;
    cameraSpeed = 100;
    zoomLevel = 2;
    particleDensity = 50;
    showMutations = true;
    showAnnotations = true;
    showTrails = false;

    dispatch('controlChange', { control: 'reset', value: true });
  }
</script>

<div class="controls-panel">
  <h3>Visualization Controls</h3>

  <!-- Color Mode -->
  <div class="control-section">
    <h4>Color Mode</h4>
    <div class="color-mode-grid">
      {#each colorModes as mode}
        <button
          class="color-mode-btn {colorMode === mode.id ? 'active' : ''}"
          on:click={() => handleColorModeChange(mode.id)}
        >
          <span class="mode-icon">{mode.icon}</span>
          <span class="mode-name">{mode.name}</span>
        </button>
      {/each}
    </div>
  </div>

  <!-- Zoom Level -->
  <div class="control-section">
    <h4>Zoom Level</h4>
    <div class="zoom-grid">
      {#each zoomLevels as level}
        <button
          class="zoom-btn {zoomLevel === level.id ? 'active' : ''}"
          on:click={() => handleZoomLevelChange(level.id)}
        >
          <span class="zoom-icon">{level.icon}</span>
          <span class="zoom-name">{level.name}</span>
          <span class="zoom-range">{level.range}</span>
        </button>
      {/each}
    </div>
  </div>

  <!-- LOD Level -->
  <div class="control-section">
    <h4>Level of Detail (LOD)</h4>
    <div class="slider-control">
      <input
        type="range"
        min="0"
        max="3"
        step="1"
        bind:value={lodLevel}
        on:input={handleLODChange}
        class="slider"
      />
      <div class="slider-labels">
        <span>Low</span>
        <span>Medium</span>
        <span>High</span>
        <span>Ultra</span>
      </div>
      <div class="slider-value">LOD Level {lodLevel}</div>
    </div>
  </div>

  <!-- Camera Speed -->
  <div class="control-section">
    <h4>Camera Speed</h4>
    <div class="slider-control">
      <input
        type="range"
        min="10"
        max="500"
        step="10"
        bind:value={cameraSpeed}
        on:input={handleCameraSpeedChange}
        class="slider"
      />
      <div class="slider-value">{cameraSpeed} units/sec</div>
    </div>
  </div>

  <!-- Particle Density -->
  <div class="control-section">
    <h4>Particle Density</h4>
    <div class="slider-control">
      <input
        type="range"
        min="1"
        max="100"
        step="1"
        bind:value={particleDensity}
        on:input={handleParticleDensityChange}
        class="slider"
      />
      <div class="slider-value">{particleDensity}%</div>
    </div>
  </div>

  <!-- Feature Toggles -->
  <div class="control-section">
    <h4>Features</h4>
    <div class="toggle-group">
      <label class="toggle-item">
        <input
          type="checkbox"
          bind:checked={showMutations}
          on:change={handleToggleMutations}
        />
        <span class="toggle-label">🔴 Show Mutations</span>
      </label>

      <label class="toggle-item">
        <input
          type="checkbox"
          bind:checked={showAnnotations}
          on:change={handleToggleAnnotations}
        />
        <span class="toggle-label">🏷️ Show Annotations</span>
      </label>

      <label class="toggle-item">
        <input
          type="checkbox"
          bind:checked={showTrails}
          on:change={handleToggleTrails}
        />
        <span class="toggle-label">✨ Particle Trails</span>
      </label>
    </div>
  </div>

  <!-- Reset Button -->
  <button class="reset-btn" on:click={resetDefaults}>
    Reset to Defaults
  </button>
</div>

<style>
  .controls-panel {
    color: #e0e0e0;
  }

  h3 {
    margin: 0 0 20px 0;
    font-size: 16px;
    font-weight: 600;
    color: #fff;
  }

  h4 {
    margin: 0 0 12px 0;
    font-size: 13px;
    font-weight: 600;
    color: #aaa;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .control-section {
    margin-bottom: 24px;
    padding-bottom: 24px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  }

  .control-section:last-of-type {
    border-bottom: none;
  }

  /* Color Mode */
  .color-mode-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .color-mode-btn {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    color: #e0e0e0;
    cursor: pointer;
    transition: all 0.2s;
    font-size: 13px;
  }

  .color-mode-btn:hover {
    background: rgba(255, 255, 255, 0.08);
    border-color: rgba(255, 255, 255, 0.2);
  }

  .color-mode-btn.active {
    background: linear-gradient(135deg, rgba(102, 126, 234, 0.2), rgba(118, 75, 162, 0.2));
    border-color: rgba(102, 126, 234, 0.5);
    color: #fff;
  }

  .mode-icon {
    font-size: 18px;
  }

  .mode-name {
    flex: 1;
  }

  /* Zoom Level */
  .zoom-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 6px;
  }

  .zoom-btn {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 12px;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    color: #e0e0e0;
    cursor: pointer;
    transition: all 0.2s;
    font-size: 12px;
  }

  .zoom-btn:hover {
    background: rgba(255, 255, 255, 0.08);
    border-color: rgba(255, 255, 255, 0.2);
  }

  .zoom-btn.active {
    background: linear-gradient(135deg, rgba(102, 126, 234, 0.2), rgba(118, 75, 162, 0.2));
    border-color: rgba(102, 126, 234, 0.5);
    color: #fff;
  }

  .zoom-icon {
    font-size: 16px;
  }

  .zoom-name {
    flex: 1;
    font-weight: 500;
  }

  .zoom-range {
    font-size: 11px;
    color: #888;
  }

  /* Sliders */
  .slider-control {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .slider {
    width: 100%;
    height: 6px;
    -webkit-appearance: none;
    appearance: none;
    background: rgba(255, 255, 255, 0.1);
    border-radius: 3px;
    outline: none;
  }

  .slider::-webkit-slider-thumb {
    -webkit-appearance: none;
    appearance: none;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: linear-gradient(135deg, #667eea, #764ba2);
    cursor: pointer;
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.3);
  }

  .slider::-moz-range-thumb {
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: linear-gradient(135deg, #667eea, #764ba2);
    cursor: pointer;
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.3);
    border: none;
  }

  .slider-labels {
    display: flex;
    justify-content: space-between;
    font-size: 10px;
    color: #666;
  }

  .slider-value {
    text-align: center;
    font-size: 13px;
    color: #aaa;
    font-weight: 500;
  }

  /* Toggles */
  .toggle-group {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .toggle-item {
    display: flex;
    align-items: center;
    gap: 10px;
    cursor: pointer;
    user-select: none;
  }

  .toggle-item input[type="checkbox"] {
    width: 18px;
    height: 18px;
    cursor: pointer;
    accent-color: #667eea;
  }

  .toggle-label {
    font-size: 13px;
    color: #e0e0e0;
  }

  /* Reset Button */
  .reset-btn {
    width: 100%;
    padding: 10px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.3);
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .reset-btn:hover {
    background: rgba(239, 68, 68, 0.2);
    border-color: rgba(239, 68, 68, 0.5);
  }
</style>
