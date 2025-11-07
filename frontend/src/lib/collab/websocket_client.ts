/**
 * GenomeVedic WebSocket Client
 * Real-time collaboration client for multiplayer genome visualization
 *
 * Features:
 * - Cursor position broadcasting (30 Hz)
 * - Viewport synchronization
 * - Follow mode
 * - Comment threads
 * - Automatic reconnection
 * - Latency tracking
 */

export type MessageType =
  | 'cursor_move'
  | 'viewport_sync'
  | 'follow_mode'
  | 'presentation_mode'
  | 'user_join'
  | 'user_leave'
  | 'user_update'
  | 'comment_add'
  | 'comment_update'
  | 'comment_delete'
  | 'mention'
  | 'heartbeat'
  | 'error'
  | 'ack';

export type Permission = 'owner' | 'editor' | 'viewer';

export interface CursorPosition {
  x: number;
  y: number;
  chromosome?: string;
  bp_position?: number;
  is_pointing?: boolean;
}

export interface ViewportState {
  chromosome: string;
  start_bp: number;
  end_bp: number;
  zoom_level: number;
  camera_x?: number;
  camera_y?: number;
  camera_z?: number;
  target_x?: number;
  target_y?: number;
  target_z?: number;
}

export interface User {
  id: string;
  name: string;
  initials: string;
  color: string;
  permission: Permission;
  cursor?: CursorPosition;
  viewport?: ViewportState;
  is_following?: string;
  joined_at: number;
  last_seen: number;
}

export interface Comment {
  id: string;
  session_id: string;
  user_id: string;
  user_name: string;
  chromosome: string;
  bp_position: number;
  content: string;
  mentions?: string[];
  parent_id?: string;
  resolved?: boolean;
  created_at: number;
  updated_at?: number;
}

export interface Message {
  id: string;
  type: MessageType;
  session_id: string;
  user_id: string;
  payload: any;
  timestamp?: number;
}

export interface SessionInfo {
  id: string;
  name: string;
  owner_id: string;
  users: Record<string, User>;
  comments?: Record<string, Comment>;
  created_at: number;
  expires_at?: number;
  is_presenting?: boolean;
  presenter_id?: string;
  max_users?: number;
}

export interface ConnectionStats {
  connected: boolean;
  latency: number;
  messagesReceived: number;
  messagesSent: number;
  reconnectAttempts: number;
}

type MessageHandler = (message: Message) => void;
type EventHandler = (data: any) => void;

export class CollabClient {
  private ws: WebSocket | null = null;
  private sessionId: string;
  private userId: string = '';
  private userName: string;
  private permission: Permission;
  private wsUrl: string;

  // Connection state
  private connected: boolean = false;
  private reconnectAttempts: number = 0;
  private maxReconnectAttempts: number = 5;
  private reconnectDelay: number = 1000; // Start with 1 second
  private reconnectTimer: number | null = null;

  // Message handlers
  private handlers: Map<MessageType, MessageHandler[]> = new Map();
  private eventHandlers: Map<string, EventHandler[]> = new Map();

  // Cursor throttling (30 Hz)
  private lastCursorSend: number = 0;
  private cursorThrottleMs: number = 33; // ~30 Hz

  // Statistics
  private stats: ConnectionStats = {
    connected: false,
    latency: 0,
    messagesReceived: 0,
    messagesSent: 0,
    reconnectAttempts: 0,
  };

  // Heartbeat
  private heartbeatInterval: number | null = null;
  private heartbeatMs: number = 30000; // 30 seconds

  constructor(
    serverUrl: string,
    sessionId: string,
    userName: string,
    permission: Permission = 'viewer'
  ) {
    this.wsUrl = `${serverUrl}/api/v1/collab/session/${sessionId}?user_name=${encodeURIComponent(userName)}&permission=${permission}`;
    this.sessionId = sessionId;
    this.userName = userName;
    this.permission = permission;
  }

  /**
   * Connect to WebSocket server
   */
  public async connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.wsUrl);

        this.ws.onopen = () => {
          console.log('[COLLAB] Connected to session:', this.sessionId);
          this.connected = true;
          this.reconnectAttempts = 0;
          this.reconnectDelay = 1000;
          this.stats.connected = true;

          this.startHeartbeat();
          this.emit('connected', {});
          resolve();
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data);
        };

        this.ws.onerror = (error) => {
          console.error('[COLLAB] WebSocket error:', error);
          this.emit('error', error);
        };

        this.ws.onclose = () => {
          console.log('[COLLAB] Disconnected from session');
          this.connected = false;
          this.stats.connected = false;
          this.stopHeartbeat();
          this.emit('disconnected', {});

          // Attempt reconnection
          this.attemptReconnect();
        };
      } catch (error) {
        reject(error);
      }
    });
  }

  /**
   * Disconnect from server
   */
  public disconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    this.stopHeartbeat();
    this.connected = false;
    this.stats.connected = false;
  }

  /**
   * Send cursor position update (throttled to 30 Hz)
   */
  public sendCursorMove(cursor: CursorPosition): void {
    const now = Date.now();
    if (now - this.lastCursorSend < this.cursorThrottleMs) {
      return; // Throttle
    }

    this.lastCursorSend = now;
    this.send({
      id: this.generateId(),
      type: 'cursor_move',
      session_id: this.sessionId,
      user_id: this.userId,
      payload: cursor,
      timestamp: now,
    });
  }

  /**
   * Send viewport synchronization
   */
  public sendViewportSync(viewport: ViewportState): void {
    this.send({
      id: this.generateId(),
      type: 'viewport_sync',
      session_id: this.sessionId,
      user_id: this.userId,
      payload: viewport,
      timestamp: Date.now(),
    });
  }

  /**
   * Enable/disable follow mode
   */
  public sendFollowMode(followUserId: string | null): void {
    this.send({
      id: this.generateId(),
      type: 'follow_mode',
      session_id: this.sessionId,
      user_id: this.userId,
      payload: { follow_user_id: followUserId },
      timestamp: Date.now(),
    });
  }

  /**
   * Enable/disable presentation mode
   */
  public sendPresentationMode(enabled: boolean): void {
    this.send({
      id: this.generateId(),
      type: 'presentation_mode',
      session_id: this.sessionId,
      user_id: this.userId,
      payload: { enabled, presenter_id: enabled ? this.userId : null },
      timestamp: Date.now(),
    });
  }

  /**
   * Add a comment
   */
  public sendComment(comment: Partial<Comment>): void {
    this.send({
      id: this.generateId(),
      type: 'comment_add',
      session_id: this.sessionId,
      user_id: this.userId,
      payload: comment,
      timestamp: Date.now(),
    });
  }

  /**
   * Update a comment
   */
  public updateComment(commentId: string, content: string): void {
    this.send({
      id: this.generateId(),
      type: 'comment_update',
      session_id: this.sessionId,
      user_id: this.userId,
      payload: { comment_id: commentId, content },
      timestamp: Date.now(),
    });
  }

  /**
   * Delete a comment
   */
  public deleteComment(commentId: string): void {
    this.send({
      id: this.generateId(),
      type: 'comment_delete',
      session_id: this.sessionId,
      user_id: this.userId,
      payload: { comment_id: commentId },
      timestamp: Date.now(),
    });
  }

  /**
   * Register message handler
   */
  public on(type: MessageType, handler: MessageHandler): void {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, []);
    }
    this.handlers.get(type)!.push(handler);
  }

  /**
   * Register event handler
   */
  public onEvent(event: string, handler: EventHandler): void {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, []);
    }
    this.eventHandlers.get(event)!.push(handler);
  }

  /**
   * Remove message handler
   */
  public off(type: MessageType, handler: MessageHandler): void {
    const handlers = this.handlers.get(type);
    if (handlers) {
      const index = handlers.indexOf(handler);
      if (index !== -1) {
        handlers.splice(index, 1);
      }
    }
  }

  /**
   * Get connection statistics
   */
  public getStats(): ConnectionStats {
    return { ...this.stats };
  }

  /**
   * Get current user ID
   */
  public getUserId(): string {
    return this.userId;
  }

  /**
   * Check if connected
   */
  public isConnected(): boolean {
    return this.connected;
  }

  // Private methods

  private send(message: Message): void {
    if (!this.connected || !this.ws) {
      console.warn('[COLLAB] Cannot send message, not connected');
      return;
    }

    try {
      this.ws.send(JSON.stringify(message));
      this.stats.messagesSent++;
    } catch (error) {
      console.error('[COLLAB] Failed to send message:', error);
    }
  }

  private handleMessage(data: string): void {
    try {
      const message: Message = JSON.parse(data);
      this.stats.messagesReceived++;

      // Calculate latency
      if (message.timestamp) {
        this.stats.latency = Date.now() - message.timestamp;
      }

      // Store user ID from first message
      if (!this.userId && message.type === 'user_join') {
        const user = message.payload as User;
        if (user.name === this.userName) {
          this.userId = user.id;
        }
      }

      // Call registered handlers
      const handlers = this.handlers.get(message.type);
      if (handlers) {
        handlers.forEach(handler => handler(message));
      }

      // Emit as event
      this.emit(message.type, message);
    } catch (error) {
      console.error('[COLLAB] Failed to parse message:', error);
    }
  }

  private emit(event: string, data: any): void {
    const handlers = this.eventHandlers.get(event);
    if (handlers) {
      handlers.forEach(handler => handler(data));
    }
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('[COLLAB] Max reconnection attempts reached');
      this.emit('reconnect_failed', {});
      return;
    }

    this.reconnectAttempts++;
    this.stats.reconnectAttempts = this.reconnectAttempts;

    console.log(`[COLLAB] Reconnecting in ${this.reconnectDelay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);

    this.reconnectTimer = window.setTimeout(() => {
      this.connect().catch(err => {
        console.error('[COLLAB] Reconnection failed:', err);
      });
    }, this.reconnectDelay);

    // Exponential backoff
    this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000); // Max 30 seconds
  }

  private startHeartbeat(): void {
    this.stopHeartbeat();

    this.heartbeatInterval = window.setInterval(() => {
      if (this.connected) {
        this.send({
          id: this.generateId(),
          type: 'heartbeat',
          session_id: this.sessionId,
          user_id: this.userId,
          payload: { timestamp: Date.now() },
          timestamp: Date.now(),
        });
      }
    }, this.heartbeatMs);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  private generateId(): string {
    return Math.random().toString(36).substring(2, 15) +
           Math.random().toString(36).substring(2, 15);
  }
}

/**
 * Create a new collaboration session
 */
export async function createSession(
  serverUrl: string,
  sessionName: string,
  userName: string,
  maxUsers: number = 100
): Promise<{ sessionId: string; userId: string; url: string }> {
  const response = await fetch(`${serverUrl}/api/v1/collab/sessions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      name: sessionName,
      user_name: userName,
      max_users: maxUsers,
    }),
  });

  if (!response.ok) {
    throw new Error(`Failed to create session: ${response.statusText}`);
  }

  const data = await response.json();
  return {
    sessionId: data.session_id,
    userId: data.user_id,
    url: data.url,
  };
}

/**
 * Get session information
 */
export async function getSessionInfo(
  serverUrl: string,
  sessionId: string
): Promise<SessionInfo> {
  const response = await fetch(`${serverUrl}/api/v1/collab/sessions/${sessionId}`);

  if (!response.ok) {
    throw new Error(`Failed to get session: ${response.statusText}`);
  }

  const data = await response.json();
  return data.session;
}
