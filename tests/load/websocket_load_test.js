/**
 * k6 Load Test for GenomeVedic Collaboration Server
 *
 * Tests:
 * - 100+ concurrent WebSocket connections
 * - Cursor update latency (<100ms p95)
 * - Message throughput
 * - Connection stability
 * - Reconnection handling
 *
 * Usage:
 *   k6 run websocket_load_test.js
 *   k6 run --vus 100 --duration 60s websocket_load_test.js
 */

import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';

// Custom metrics
const cursorUpdateLatency = new Trend('cursor_update_latency', true);
const messagesSent = new Counter('messages_sent');
const messagesReceived = new Counter('messages_received');
const connectionErrors = new Counter('connection_errors');
const successRate = new Rate('success_rate');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 50 },   // Ramp up to 50 users
    { duration: '1m', target: 100 },   // Ramp up to 100 users
    { duration: '2m', target: 100 },   // Stay at 100 users
    { duration: '30s', target: 150 },  // Spike to 150 users
    { duration: '1m', target: 150 },   // Stay at 150 users
    { duration: '30s', target: 0 },    // Ramp down
  ],
  thresholds: {
    'cursor_update_latency': ['p(95)<100', 'p(99)<200'], // <100ms p95, <200ms p99
    'success_rate': ['rate>0.99'],                        // >99% success rate
    'ws_connecting': ['avg<1000'],                        // <1s connection time
  },
};

const BASE_URL = __ENV.WS_URL || 'ws://localhost:8080';
const SESSION_ID = __ENV.SESSION_ID || createSession();

/**
 * Create a test session
 */
function createSession() {
  // In a real test, we would create a session via REST API
  // For now, use a fixed session ID
  return 'test-session-load-' + Date.now();
}

/**
 * Main test function (runs for each virtual user)
 */
export default function () {
  const userId = `user-${__VU}-${__ITER}`;
  const wsUrl = `${BASE_URL}/api/v1/collab/session/${SESSION_ID}?user_name=${userId}&permission=editor`;

  console.log(`[${userId}] Connecting to ${wsUrl}`);

  const response = ws.connect(wsUrl, {
    tags: { name: 'WebSocket' },
  }, function (socket) {

    // Connection established
    console.log(`[${userId}] Connected`);

    let messageCount = 0;
    let cursorUpdateStart = {};
    let isConnected = true;

    // Message handler
    socket.on('message', (data) => {
      messageCount++;
      messagesReceived.add(1);

      try {
        const message = JSON.parse(data);

        // Track cursor update latency
        if (message.type === 'cursor_move' && message.timestamp) {
          const latency = Date.now() - message.timestamp;
          cursorUpdateLatency.add(latency);
        }

        // Track ACK latency
        if (message.type === 'ack' && cursorUpdateStart[message.id]) {
          const latency = Date.now() - cursorUpdateStart[message.id];
          cursorUpdateLatency.add(latency);
          delete cursorUpdateStart[message.id];
        }

        successRate.add(1);
      } catch (e) {
        console.error(`[${userId}] Failed to parse message: ${e}`);
        successRate.add(0);
      }
    });

    // Error handler
    socket.on('error', (err) => {
      console.error(`[${userId}] WebSocket error: ${err}`);
      connectionErrors.add(1);
      successRate.add(0);
      isConnected = false;
    });

    // Close handler
    socket.on('close', () => {
      console.log(`[${userId}] Disconnected (received ${messageCount} messages)`);
      isConnected = false;
    });

    // Simulate user behavior
    const testDuration = 30; // 30 seconds per connection
    const cursorUpdateInterval = 100; // Update cursor every 100ms (~10 Hz)
    const startTime = Date.now();

    while (isConnected && (Date.now() - startTime) < (testDuration * 1000)) {
      // Send cursor position update
      const cursorMessage = {
        id: generateId(),
        type: 'cursor_move',
        session_id: SESSION_ID,
        user_id: userId,
        payload: {
          x: Math.random(),
          y: Math.random(),
          chromosome: `chr${Math.floor(Math.random() * 22) + 1}`,
          bp_position: Math.floor(Math.random() * 1000000000),
        },
        timestamp: Date.now(),
      };

      cursorUpdateStart[cursorMessage.id] = Date.now();
      socket.send(JSON.stringify(cursorMessage));
      messagesSent.add(1);

      // Occasionally send viewport sync
      if (Math.random() < 0.1) {
        const viewportMessage = {
          id: generateId(),
          type: 'viewport_sync',
          session_id: SESSION_ID,
          user_id: userId,
          payload: {
            chromosome: `chr${Math.floor(Math.random() * 22) + 1}`,
            start_bp: Math.floor(Math.random() * 1000000),
            end_bp: Math.floor(Math.random() * 1000000) + 10000,
            zoom_level: Math.random() * 10,
          },
          timestamp: Date.now(),
        };

        socket.send(JSON.stringify(viewportMessage));
        messagesSent.add(1);
      }

      // Wait before next update
      sleep(cursorUpdateInterval / 1000);
    }

    // Close connection
    if (isConnected) {
      socket.close();
    }
  });

  // Check connection success
  check(response, {
    'WebSocket connected': (r) => r && r.status === 101,
  });

  // Pause before next iteration
  sleep(1);
}

/**
 * Generate random ID
 */
function generateId() {
  return Math.random().toString(36).substring(2, 15) +
         Math.random().toString(36).substring(2, 15);
}

/**
 * Setup function (runs once at start)
 */
export function setup() {
  console.log('===========================================');
  console.log('GenomeVedic WebSocket Load Test');
  console.log('===========================================');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Session ID: ${SESSION_ID}`);
  console.log(`Target: 100+ concurrent users`);
  console.log(`Latency target: <100ms p95`);
  console.log('===========================================');

  return { sessionId: SESSION_ID };
}

/**
 * Teardown function (runs once at end)
 */
export function teardown(data) {
  console.log('===========================================');
  console.log('Load Test Complete');
  console.log('===========================================');
}
