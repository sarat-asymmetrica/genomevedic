<script>
  /**
   * Stats Panel Component
   *
   * Displays real-time performance metrics:
   * - FPS (frames per second)
   * - Frame time (ms)
   * - Particle counts
   * - Memory usage
   */

  export let stats = {
    fps: 0,
    frameTime: 0,
    particleCount: 0,
    visibleParticles: 0,
    memoryUsage: 0
  };

  // Performance thresholds
  const FPS_GOOD = 60;
  const FPS_OK = 30;

  $: fpsStatus = stats.fps >= FPS_GOOD ? 'good' : stats.fps >= FPS_OK ? 'ok' : 'bad';
  $: frameTimeStatus = stats.frameTime <= 16.67 ? 'good' : stats.frameTime <= 33.33 ? 'ok' : 'bad';
</script>

<div class="stats-panel">
  <h3>Performance Stats</h3>

  <div class="stats-grid">
    <!-- FPS -->
    <div class="stat-item">
      <div class="stat-label">FPS</div>
      <div class="stat-value {fpsStatus}">
        {stats.fps}
      </div>
      <div class="stat-bar">
        <div class="stat-bar-fill {fpsStatus}" style="width: {Math.min(stats.fps / 120 * 100, 100)}%"></div>
      </div>
    </div>

    <!-- Frame Time -->
    <div class="stat-item">
      <div class="stat-label">Frame Time</div>
      <div class="stat-value {frameTimeStatus}">
        {stats.frameTime.toFixed(2)} ms
      </div>
      <div class="stat-bar">
        <div class="stat-bar-fill {frameTimeStatus}" style="width: {Math.min((33.33 - stats.frameTime) / 33.33 * 100, 100)}%"></div>
      </div>
    </div>

    <!-- Particle Count -->
    <div class="stat-item">
      <div class="stat-label">Total Particles</div>
      <div class="stat-value">
        {stats.particleCount.toLocaleString()}
      </div>
    </div>

    <!-- Visible Particles -->
    <div class="stat-item">
      <div class="stat-label">Visible Particles</div>
      <div class="stat-value">
        {stats.visibleParticles.toLocaleString()}
      </div>
      <div class="stat-sublabel">
        {((stats.visibleParticles / stats.particleCount) * 100).toFixed(1)}% visible
      </div>
    </div>

    <!-- Memory Usage -->
    <div class="stat-item">
      <div class="stat-label">Memory Usage</div>
      <div class="stat-value">
        {(stats.memoryUsage / 1024 / 1024).toFixed(2)} MB
      </div>
    </div>
  </div>
</div>

<style>
  .stats-panel {
    color: #e0e0e0;
  }

  h3 {
    margin: 0 0 16px 0;
    font-size: 16px;
    font-weight: 600;
    color: #fff;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
    gap: 16px;
  }

  .stat-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .stat-label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: #888;
    font-weight: 500;
  }

  .stat-value {
    font-size: 20px;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }

  .stat-value.good {
    color: #4ade80;
  }

  .stat-value.ok {
    color: #fbbf24;
  }

  .stat-value.bad {
    color: #f87171;
  }

  .stat-sublabel {
    font-size: 11px;
    color: #666;
  }

  .stat-bar {
    width: 100%;
    height: 4px;
    background: rgba(255, 255, 255, 0.1);
    border-radius: 2px;
    overflow: hidden;
    margin-top: 4px;
  }

  .stat-bar-fill {
    height: 100%;
    transition: width 0.3s ease;
    border-radius: 2px;
  }

  .stat-bar-fill.good {
    background: linear-gradient(90deg, #4ade80, #22c55e);
  }

  .stat-bar-fill.ok {
    background: linear-gradient(90deg, #fbbf24, #f59e0b);
  }

  .stat-bar-fill.bad {
    background: linear-gradient(90deg, #f87171, #ef4444);
  }
</style>
