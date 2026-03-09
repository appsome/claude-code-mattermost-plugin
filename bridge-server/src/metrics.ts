/**
 * Prometheus Metrics Module
 * 
 * Exposes application metrics for monitoring and alerting
 */

import { Counter, Histogram, Gauge, register } from 'prom-client';

// Enable default metrics (CPU, memory, event loop, etc.)
register.setDefaultLabels({
  app: 'claude-code-bridge',
});

// Session metrics
export const sessionCounter = new Counter({
  name: 'claude_code_sessions_total',
  help: 'Total number of Claude Code sessions created',
  labelNames: ['status'] as const,
});

export const activeSessionsGauge = new Gauge({
  name: 'claude_code_sessions_active',
  help: 'Number of currently active Claude Code sessions',
});

// Message metrics
export const messageCounter = new Counter({
  name: 'claude_code_messages_total',
  help: 'Total number of messages processed',
  labelNames: ['direction'] as const, // 'inbound' or 'outbound'
});

export const messageHistogram = new Histogram({
  name: 'claude_code_message_duration_seconds',
  help: 'Message processing time in seconds',
  buckets: [0.1, 0.5, 1, 2, 5, 10, 30, 60],
});

// WebSocket metrics
export const wsConnectionsGauge = new Gauge({
  name: 'claude_code_websocket_connections',
  help: 'Number of active WebSocket connections',
});

export const wsMessageCounter = new Counter({
  name: 'claude_code_websocket_messages_total',
  help: 'Total number of WebSocket messages',
  labelNames: ['type'] as const, // 'received' or 'sent'
});

// API endpoint metrics
export const httpRequestCounter = new Counter({
  name: 'claude_code_http_requests_total',
  help: 'Total number of HTTP requests',
  labelNames: ['method', 'path', 'status'] as const,
});

export const httpRequestDuration = new Histogram({
  name: 'claude_code_http_request_duration_seconds',
  help: 'HTTP request duration in seconds',
  labelNames: ['method', 'path'] as const,
  buckets: [0.01, 0.05, 0.1, 0.5, 1, 2, 5],
});

// Error metrics
export const errorCounter = new Counter({
  name: 'claude_code_errors_total',
  help: 'Total number of errors',
  labelNames: ['type'] as const,
});

// Database metrics
export const dbQueryCounter = new Counter({
  name: 'claude_code_db_queries_total',
  help: 'Total number of database queries',
  labelNames: ['operation'] as const,
});

export const dbQueryDuration = new Histogram({
  name: 'claude_code_db_query_duration_seconds',
  help: 'Database query duration in seconds',
  labelNames: ['operation'] as const,
  buckets: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1],
});

// File operation metrics
export const fileOperationCounter = new Counter({
  name: 'claude_code_file_operations_total',
  help: 'Total number of file operations',
  labelNames: ['operation'] as const, // 'read', 'write', 'list', etc.
});

// Export the registry for /metrics endpoint
export { register };
