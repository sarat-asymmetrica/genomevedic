<script>
  /**
   * CollaboratorCursors Component
   * Displays real-time cursor positions of all collaborators
   *
   * Features:
   * - Smooth cursor interpolation
   * - Avatar with initials and color
   * - User name tooltip
   * - Cursor pointer animation
   * - 60 fps smooth rendering
   */

  import { onMount, onDestroy } from 'svelte';

  // Props
  export let users = {}; // Record<string, User>
  export let currentUserId = '';
  export let canvasElement = null; // Reference to main canvas

  // State
  let cursors = new Map(); // userId -> {x, y, displayX, displayY, lastUpdate}
  let animationFrameId;

  // Animation constants
  const LERP_FACTOR = 0.3; // Smoothing factor (0-1)
  const CURSOR_FADEOUT_MS = 5000; // Hide cursor after 5s of inactivity

  onMount(() => {
    startAnimation();
  });

  onDestroy(() => {
    if (animationFrameId) {
      cancelAnimationFrame(animationFrameId);
    }
  });

  // Update cursor positions from user data
  $: {
    Object.keys(users).forEach(userId => {
      if (userId === currentUserId) return; // Don't show own cursor

      const user = users[userId];
      if (user.cursor) {
        updateCursor(userId, user.cursor, user);
      }
    });
  }

  function updateCursor(userId, cursor, user) {
    const now = Date.now();

    if (!cursors.has(userId)) {
      // Initialize cursor
      cursors.set(userId, {
        x: cursor.x,
        y: cursor.y,
        displayX: cursor.x,
        displayY: cursor.y,
        lastUpdate: now,
        user: user,
      });
    } else {
      // Update target position
      const cursorData = cursors.get(userId);
      cursorData.x = cursor.x;
      cursorData.y = cursor.y;
      cursorData.lastUpdate = now;
      cursorData.user = user;
      cursors.set(userId, cursorData);
    }

    // Trigger reactivity
    cursors = cursors;
  }

  function startAnimation() {
    function animate() {
      const now = Date.now();

      // Interpolate cursor positions
      cursors.forEach((cursorData, userId) => {
        // Smooth interpolation (lerp)
        cursorData.displayX += (cursorData.x - cursorData.displayX) * LERP_FACTOR;
        cursorData.displayY += (cursorData.y - cursorData.displayY) * LERP_FACTOR;

        // Check if cursor should be hidden (inactive)
        const timeSinceUpdate = now - cursorData.lastUpdate;
        if (timeSinceUpdate > CURSOR_FADEOUT_MS) {
          cursors.delete(userId);
        }
      });

      // Trigger reactivity
      cursors = cursors;

      animationFrameId = requestAnimationFrame(animate);
    }

    animate();
  }

  function getCursorStyle(cursorData) {
    // Get canvas dimensions
    const rect = canvasElement?.getBoundingClientRect() || { width: window.innerWidth, height: window.innerHeight };

    // Convert normalized coordinates (0-1) to screen pixels
    const x = cursorData.displayX * rect.width;
    const y = cursorData.displayY * rect.height + (canvasElement?.offsetTop || 60);

    return `left: ${x}px; top: ${y}px;`;
  }

  function getAvatarStyle(user) {
    return `background-color: ${user.color};`;
  }
</script>

<!-- Render cursors -->
{#each Array.from(cursors.entries()) as [userId, cursorData]}
  <div class="collaborator-cursor" style={getCursorStyle(cursorData)}>
    <!-- Cursor pointer -->
    <svg class="cursor-pointer" width="24" height="24" viewBox="0 0 24 24" style="fill: {cursorData.user.color};">
      <path d="M5.5 3.21V20.79L9.82 16.47L12.5 21L15.5 19.5L12.82 14.97L18.5 14.5L5.5 3.21Z" />
    </svg>

    <!-- Avatar badge -->
    <div class="cursor-avatar" style={getAvatarStyle(cursorData.user)}>
      <span class="avatar-initials">{cursorData.user.initials}</span>
    </div>

    <!-- User name tooltip -->
    <div class="cursor-tooltip">{cursorData.user.name}</div>
  </div>
{/each}

<style>
  .collaborator-cursor {
    position: absolute;
    pointer-events: none;
    z-index: 9999;
    transition: opacity 0.3s ease;
  }

  .cursor-pointer {
    filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.3));
  }

  .cursor-avatar {
    position: absolute;
    top: 20px;
    left: 12px;
    width: 28px;
    height: 28px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 2px solid rgba(255, 255, 255, 0.9);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  }

  .avatar-initials {
    color: white;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
  }

  .cursor-tooltip {
    position: absolute;
    top: 20px;
    left: 45px;
    background: rgba(20, 20, 20, 0.95);
    color: white;
    padding: 4px 10px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 500;
    white-space: nowrap;
    pointer-events: none;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .collaborator-cursor:hover .cursor-tooltip {
    opacity: 1;
  }
</style>
