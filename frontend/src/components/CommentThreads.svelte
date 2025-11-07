<script>
  /**
   * CommentThreads Component
   * Real-time comment threads for genomic positions
   *
   * Features:
   * - Click variant to add comment
   * - Markdown support (bold, italic, links, code)
   * - @mentions with autocomplete
   * - Real-time updates
   * - Nested replies
   * - Resolve/unresolve threads
   */

  import { onMount } from 'svelte';

  // Props
  export let comments = {}; // Record<string, Comment>
  export let users = {}; // Record<string, User>
  export let currentUserId = '';
  export let canEdit = true;

  // Callbacks
  export let onAddComment = (comment) => {};
  export let onUpdateComment = (commentId, content) => {};
  export let onDeleteComment = (commentId) => {};

  // State
  let showCommentPanel = false;
  let newCommentContent = '';
  let selectedPosition = null;
  let replyToComment = null;
  let filterResolved = false;

  // Get sorted comments
  $: sortedComments = Object.values(comments)
    .filter(c => !c.parent_id) // Top-level comments only
    .filter(c => !filterResolved || !c.resolved)
    .sort((a, b) => b.created_at - a.created_at);

  function addComment() {
    if (!newCommentContent.trim()) return;

    const comment = {
      chromosome: selectedPosition?.chromosome || 'chr1',
      bp_position: selectedPosition?.bp_position || 0,
      content: newCommentContent,
      parent_id: replyToComment?.id || undefined,
      mentions: extractMentions(newCommentContent),
    };

    onAddComment(comment);

    // Reset form
    newCommentContent = '';
    replyToComment = null;
  }

  function extractMentions(content) {
    const mentions = [];
    const regex = /@(\w+)/g;
    let match;

    while ((match = regex.exec(content)) !== null) {
      mentions.push(match[1]);
    }

    return mentions;
  }

  function renderMarkdown(content) {
    // Simple markdown rendering
    let html = content;

    // Bold: **text** or __text__
    html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
    html = html.replace(/__(.+?)__/g, '<strong>$1</strong>');

    // Italic: *text* or _text_
    html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');
    html = html.replace(/_(.+?)_/g, '<em>$1</em>');

    // Code: `code`
    html = html.replace(/`(.+?)`/g, '<code>$1</code>');

    // Links: [text](url)
    html = html.replace(/\[(.+?)\]\((.+?)\)/g, '<a href="$2" target="_blank">$1</a>');

    // Mentions: @username
    html = html.replace(/@(\w+)/g, '<span class="mention">@$1</span>');

    return html;
  }

  function formatTime(timestamp) {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return 'just now';
    if (diffMins < 60) return `${diffMins}m ago`;

    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;

    const diffDays = Math.floor(diffHours / 24);
    return `${diffDays}d ago`;
  }

  function getUser(userId) {
    return users[userId] || { name: 'Unknown', color: '#888', initials: '??' };
  }

  function getReplies(commentId) {
    return Object.values(comments).filter(c => c.parent_id === commentId);
  }

  function toggleResolve(comment) {
    onUpdateComment(comment.id, comment.content);
    // Server should toggle resolved status
  }

  function replyTo(comment) {
    replyToComment = comment;
    showCommentPanel = true;
  }
</script>

<div class="comment-panel" class:open={showCommentPanel}>
  <!-- Header -->
  <div class="panel-header">
    <h3>Comments</h3>
    <div class="header-actions">
      <label class="filter-toggle">
        <input type="checkbox" bind:checked={filterResolved} />
        Hide resolved
      </label>
      <button class="icon-btn" on:click={() => showCommentPanel = !showCommentPanel}>
        {showCommentPanel ? '✕' : '💬'}
      </button>
    </div>
  </div>

  <!-- Comment list -->
  <div class="comment-list">
    {#if sortedComments.length === 0}
      <div class="empty-state">
        <p>No comments yet</p>
        <p class="hint">Click on a variant to add a comment</p>
      </div>
    {:else}
      {#each sortedComments as comment}
        <div class="comment-thread" class:resolved={comment.resolved}>
          <!-- Main comment -->
          <div class="comment">
            <div class="comment-avatar" style="background-color: {getUser(comment.user_id).color}">
              {getUser(comment.user_id).initials}
            </div>

            <div class="comment-content">
              <div class="comment-header">
                <span class="comment-author">{comment.user_name}</span>
                <span class="comment-position">
                  {comment.chromosome}:{comment.bp_position.toLocaleString()}
                </span>
                <span class="comment-time">{formatTime(comment.created_at)}</span>
              </div>

              <div class="comment-body">
                {@html renderMarkdown(comment.content)}
              </div>

              <div class="comment-actions">
                {#if canEdit}
                  <button class="action-btn" on:click={() => replyTo(comment)}>Reply</button>
                  <button class="action-btn" on:click={() => toggleResolve(comment)}>
                    {comment.resolved ? 'Reopen' : 'Resolve'}
                  </button>
                {/if}
              </div>
            </div>
          </div>

          <!-- Replies -->
          {#each getReplies(comment.id) as reply}
            <div class="comment reply">
              <div class="comment-avatar" style="background-color: {getUser(reply.user_id).color}">
                {getUser(reply.user_id).initials}
              </div>

              <div class="comment-content">
                <div class="comment-header">
                  <span class="comment-author">{reply.user_name}</span>
                  <span class="comment-time">{formatTime(reply.created_at)}</span>
                </div>

                <div class="comment-body">
                  {@html renderMarkdown(reply.content)}
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/each}
    {/if}
  </div>

  <!-- New comment form -->
  {#if canEdit}
    <div class="comment-form">
      {#if replyToComment}
        <div class="reply-context">
          Replying to {getUser(replyToComment.user_id).name}
          <button class="clear-btn" on:click={() => replyToComment = null}>✕</button>
        </div>
      {/if}

      <textarea
        bind:value={newCommentContent}
        placeholder="Add a comment... (Markdown supported, use @ to mention)"
        rows="3"
      ></textarea>

      <div class="form-actions">
        <button class="btn-primary" on:click={addComment} disabled={!newCommentContent.trim()}>
          {replyToComment ? 'Reply' : 'Comment'}
        </button>
        <span class="markdown-hint">Markdown supported</span>
      </div>
    </div>
  {/if}
</div>

<style>
  .comment-panel {
    position: fixed;
    top: 60px;
    right: 0;
    width: 400px;
    height: calc(100vh - 60px);
    background: rgba(20, 20, 20, 0.98);
    border-left: 1px solid rgba(255, 255, 255, 0.1);
    display: flex;
    flex-direction: column;
    transform: translateX(100%);
    transition: transform 0.3s ease;
    z-index: 90;
  }

  .comment-panel.open {
    transform: translateX(0);
  }

  .panel-header {
    padding: 20px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .panel-header h3 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: #e0e0e0;
  }

  .header-actions {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .filter-toggle {
    font-size: 13px;
    color: #a0a0a0;
    display: flex;
    align-items: center;
    gap: 6px;
    cursor: pointer;
  }

  .icon-btn {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: #e0e0e0;
    padding: 6px 10px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 16px;
    transition: all 0.2s;
  }

  .icon-btn:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  .comment-list {
    flex: 1;
    overflow-y: auto;
    padding: 20px;
  }

  .empty-state {
    text-align: center;
    color: #888;
    padding: 60px 20px;
  }

  .empty-state p {
    margin: 8px 0;
  }

  .hint {
    font-size: 13px;
  }

  .comment-thread {
    margin-bottom: 24px;
    opacity: 1;
    transition: opacity 0.2s;
  }

  .comment-thread.resolved {
    opacity: 0.6;
  }

  .comment {
    display: flex;
    gap: 12px;
    margin-bottom: 12px;
  }

  .comment.reply {
    margin-left: 40px;
  }

  .comment-avatar {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    font-size: 12px;
    font-weight: 600;
    flex-shrink: 0;
  }

  .comment-content {
    flex: 1;
    min-width: 0;
  }

  .comment-header {
    display: flex;
    gap: 8px;
    align-items: center;
    margin-bottom: 6px;
    flex-wrap: wrap;
  }

  .comment-author {
    font-weight: 600;
    color: #e0e0e0;
    font-size: 13px;
  }

  .comment-position {
    font-size: 12px;
    color: #667eea;
    font-family: 'Courier New', monospace;
  }

  .comment-time {
    font-size: 12px;
    color: #888;
    margin-left: auto;
  }

  .comment-body {
    color: #c0c0c0;
    font-size: 14px;
    line-height: 1.6;
    word-wrap: break-word;
  }

  .comment-body :global(strong) {
    font-weight: 600;
    color: #e0e0e0;
  }

  .comment-body :global(code) {
    background: rgba(255, 255, 255, 0.1);
    padding: 2px 6px;
    border-radius: 3px;
    font-family: 'Courier New', monospace;
    font-size: 13px;
  }

  .comment-body :global(.mention) {
    color: #667eea;
    font-weight: 500;
  }

  .comment-body :global(a) {
    color: #667eea;
    text-decoration: none;
  }

  .comment-body :global(a:hover) {
    text-decoration: underline;
  }

  .comment-actions {
    display: flex;
    gap: 12px;
    margin-top: 8px;
  }

  .action-btn {
    background: none;
    border: none;
    color: #888;
    font-size: 12px;
    cursor: pointer;
    padding: 4px 0;
    transition: color 0.2s;
  }

  .action-btn:hover {
    color: #e0e0e0;
  }

  .comment-form {
    border-top: 1px solid rgba(255, 255, 255, 0.1);
    padding: 16px 20px;
  }

  .reply-context {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 12px;
    background: rgba(102, 126, 234, 0.1);
    border-left: 3px solid #667eea;
    border-radius: 4px;
    margin-bottom: 12px;
    font-size: 13px;
    color: #c0c0c0;
  }

  .clear-btn {
    background: none;
    border: none;
    color: #888;
    cursor: pointer;
    padding: 0;
    font-size: 16px;
  }

  textarea {
    width: 100%;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    padding: 12px;
    color: #e0e0e0;
    font-size: 14px;
    font-family: inherit;
    resize: vertical;
    min-height: 60px;
  }

  textarea:focus {
    outline: none;
    border-color: #667eea;
    background: rgba(255, 255, 255, 0.08);
  }

  .form-actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 12px;
  }

  .btn-primary {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    border: none;
    color: white;
    padding: 8px 20px;
    border-radius: 6px;
    cursor: pointer;
    font-weight: 500;
    font-size: 14px;
    transition: transform 0.2s, opacity 0.2s;
  }

  .btn-primary:hover:not(:disabled) {
    transform: translateY(-1px);
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .markdown-hint {
    font-size: 12px;
    color: #888;
  }
</style>
