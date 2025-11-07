<script>
  /**
   * SessionManager Component
   * Manages collaboration session controls and user list
   *
   * Features:
   * - Session URL sharing
   * - Active users list
   * - Follow mode toggle
   * - Presentation mode (owner only)
   * - Connection status
   * - Latency display
   */

  import { onMount } from 'svelte';

  // Props
  export let sessionId = '';
  export let sessionUrl = '';
  export let users = {}; // Record<string, User>
  export let currentUserId = '';
  export let stats = null; // ConnectionStats
  export let isOwner = false;

  // Callbacks
  export let onFollowUser = (userId) => {};
  export let onUnfollow = () => {};
  export let onTogglePresentation = (enabled) => {};

  // State
  let showPanel = false;
  let followingUserId = null;
  let isPresentationMode = false;
  let copiedUrl = false;

  $: activeUsers = Object.values(users).filter(u => u.id !== currentUserId);
  $: currentUser = users[currentUserId];

  function copySessionUrl() {
    navigator.clipboard.writeText(sessionUrl).then(() => {
      copiedUrl = true;
      setTimeout(() => copiedUrl = false, 2000);
    });
  }

  function followUser(userId) {
    if (followingUserId === userId) {
      followingUserId = null;
      onUnfollow();
    } else {
      followingUserId = userId;
      onFollowUser(userId);
    }
  }

  function togglePresentation() {
    isPresentationMode = !isPresentationMode;
    onTogglePresentation(isPresentationMode);
  }

  function getConnectionQuality(latency) {
    if (latency < 50) return { label: 'Excellent', color: '#43e97b' };
    if (latency < 100) return { label: 'Good', color: '#ffa647' };
    if (latency < 200) return { label: 'Fair', color: '#fe8c00' };
    return { label: 'Poor', color: '#f83600' };
  }

  $: connectionQuality = stats ? getConnectionQuality(stats.latency) : null;
</script>

<div class="session-manager">
  <!-- Trigger button -->
  <button class="trigger-btn" on:click={() => showPanel = !showPanel} title="Session Controls">
    <span class="user-count">{Object.keys(users).length}</span>
    <span class="icon">👥</span>
  </button>

  <!-- Panel -->
  {#if showPanel}
    <div class="session-panel">
      <!-- Session info -->
      <div class="panel-section">
        <h4>Session</h4>
        <div class="session-url-box">
          <input
            type="text"
            value={sessionUrl}
            readonly
            class="url-input"
          />
          <button class="copy-btn" on:click={copySessionUrl}>
            {copiedUrl ? '✓' : '📋'}
          </button>
        </div>
        {#if copiedUrl}
          <div class="copied-toast">Link copied!</div>
        {/if}
      </div>

      <!-- Connection status -->
      {#if stats}
        <div class="panel-section">
          <h4>Connection</h4>
          <div class="connection-stats">
            <div class="stat-row">
              <span class="stat-label">Status:</span>
              <span class="stat-value" class:connected={stats.connected}>
                {stats.connected ? 'Connected' : 'Disconnected'}
              </span>
            </div>
            <div class="stat-row">
              <span class="stat-label">Latency:</span>
              <span class="stat-value" style="color: {connectionQuality?.color}">
                {stats.latency}ms ({connectionQuality?.label})
              </span>
            </div>
            <div class="stat-row">
              <span class="stat-label">Messages:</span>
              <span class="stat-value">
                ↓{stats.messagesReceived} ↑{stats.messagesSent}
              </span>
            </div>
          </div>
        </div>
      {/if}

      <!-- Presentation mode (owner only) -->
      {#if isOwner}
        <div class="panel-section">
          <h4>Presentation Mode</h4>
          <label class="toggle-switch">
            <input
              type="checkbox"
              checked={isPresentationMode}
              on:change={togglePresentation}
            />
            <span class="toggle-slider"></span>
            <span class="toggle-label">
              {isPresentationMode ? 'On - You control all views' : 'Off'}
            </span>
          </label>
        </div>
      {/if}

      <!-- Active users -->
      <div class="panel-section">
        <h4>Active Users ({activeUsers.length + 1})</h4>

        <!-- Current user -->
        {#if currentUser}
          <div class="user-item you">
            <div class="user-avatar" style="background-color: {currentUser.color}">
              {currentUser.initials}
            </div>
            <div class="user-info">
              <div class="user-name">{currentUser.name} (You)</div>
              <div class="user-role">{currentUser.permission}</div>
            </div>
          </div>
        {/if}

        <!-- Other users -->
        {#each activeUsers as user}
          <div class="user-item">
            <div class="user-avatar" style="background-color: {user.color}">
              {user.initials}
            </div>
            <div class="user-info">
              <div class="user-name">{user.name}</div>
              <div class="user-role">{user.permission}</div>
            </div>
            <button
              class="follow-btn"
              class:active={followingUserId === user.id}
              on:click={() => followUser(user.id)}
              title={followingUserId === user.id ? 'Stop following' : 'Follow view'}
            >
              {followingUserId === user.id ? '👁️' : '👁'}
            </button>
          </div>
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .session-manager {
    position: relative;
  }

  .trigger-btn {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: #e0e0e0;
    padding: 8px 12px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 16px;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: all 0.2s;
  }

  .trigger-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    border-color: rgba(255, 255, 255, 0.2);
    transform: translateY(-1px);
  }

  .user-count {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    font-size: 12px;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 10px;
    min-width: 20px;
    text-align: center;
  }

  .session-panel {
    position: absolute;
    top: 50px;
    right: 0;
    width: 320px;
    max-height: 600px;
    overflow-y: auto;
    background: rgba(20, 20, 20, 0.98);
    backdrop-filter: blur(10px);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
    z-index: 1000;
  }

  .panel-section {
    padding: 16px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .panel-section:last-child {
    border-bottom: none;
  }

  .panel-section h4 {
    margin: 0 0 12px 0;
    font-size: 13px;
    font-weight: 600;
    color: #888;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .session-url-box {
    display: flex;
    gap: 8px;
    margin-bottom: 8px;
  }

  .url-input {
    flex: 1;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    padding: 8px 12px;
    color: #c0c0c0;
    font-size: 12px;
    font-family: 'Courier New', monospace;
  }

  .copy-btn {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: #e0e0e0;
    padding: 8px 12px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 16px;
    transition: all 0.2s;
  }

  .copy-btn:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  .copied-toast {
    background: #43e97b;
    color: white;
    padding: 6px 12px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 500;
    text-align: center;
    animation: fadeIn 0.2s ease;
  }

  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(-4px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .connection-stats {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .stat-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 13px;
  }

  .stat-label {
    color: #888;
  }

  .stat-value {
    color: #e0e0e0;
    font-weight: 500;
    font-family: 'Courier New', monospace;
  }

  .stat-value.connected {
    color: #43e97b;
  }

  .toggle-switch {
    display: flex;
    align-items: center;
    gap: 12px;
    cursor: pointer;
  }

  .toggle-switch input {
    display: none;
  }

  .toggle-slider {
    position: relative;
    width: 44px;
    height: 24px;
    background: rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    transition: background 0.2s;
  }

  .toggle-slider::after {
    content: '';
    position: absolute;
    top: 2px;
    left: 2px;
    width: 20px;
    height: 20px;
    background: white;
    border-radius: 50%;
    transition: transform 0.2s;
  }

  .toggle-switch input:checked + .toggle-slider {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  }

  .toggle-switch input:checked + .toggle-slider::after {
    transform: translateX(20px);
  }

  .toggle-label {
    font-size: 13px;
    color: #c0c0c0;
  }

  .user-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px;
    background: rgba(255, 255, 255, 0.03);
    border-radius: 6px;
    margin-bottom: 8px;
    transition: background 0.2s;
  }

  .user-item:hover {
    background: rgba(255, 255, 255, 0.06);
  }

  .user-item.you {
    background: rgba(102, 126, 234, 0.1);
    border: 1px solid rgba(102, 126, 234, 0.3);
  }

  .user-avatar {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    font-size: 14px;
    font-weight: 600;
    flex-shrink: 0;
  }

  .user-info {
    flex: 1;
    min-width: 0;
  }

  .user-name {
    font-size: 14px;
    font-weight: 500;
    color: #e0e0e0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .user-role {
    font-size: 12px;
    color: #888;
    text-transform: capitalize;
  }

  .follow-btn {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: #e0e0e0;
    padding: 6px 10px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 16px;
    transition: all 0.2s;
  }

  .follow-btn:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  .follow-btn.active {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    border-color: transparent;
  }
</style>
