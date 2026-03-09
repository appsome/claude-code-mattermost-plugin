/**
 * Bridge Server Entry Point
 * 
 * Provides REST API + WebSocket interface for managing Claude Code CLI sessions
 * Used by the Mattermost plugin to control Claude Code remotely
 */

import express, { Request, Response, NextFunction } from 'express';
import cors from 'cors';
import { createServer } from 'http';
import { WebSocketServer } from 'ws';
import { config } from './config.js';
import { logger } from './logger.js';
import { db } from './database.js';
import { spawner } from './spawner.js';
import WebSocketHandler from './websocket/handler.js';
import { 
  register, 
  httpRequestCounter, 
  httpRequestDuration,
  activeSessionsGauge 
} from './metrics.js';

// Import API routers
import sessionsRouter from './api/sessions.js';
import messagesRouter from './api/messages.js';
import filesRouter from './api/files.js';
import gitRouter from './api/git.js';
import contextRouter from './api/context.js';

// Initialize Express app
const app = express();

// Middleware
app.use(cors({
  origin: config.corsOrigin,
  credentials: true,
}));
app.use(express.json());
app.use(express.urlencoded({ extended: true }));

// Request logging and metrics
app.use((req: Request, res: Response, next: NextFunction) => {
  const start = Date.now();
  logger.debug(`${req.method} ${req.path}`);
  
  // Track response to record metrics
  res.on('finish', () => {
    const duration = (Date.now() - start) / 1000;
    httpRequestCounter.inc({
      method: req.method,
      path: req.route?.path || req.path,
      status: res.statusCode.toString(),
    });
    httpRequestDuration.observe({
      method: req.method,
      path: req.route?.path || req.path,
    }, duration);
  });
  
  next();
});

// Track server start time for uptime calculation
const startTime = Date.now();

// Health check endpoint
app.get('/health', (req: Request, res: Response) => {
  const activeSessions = spawner.getAllProcesses().length;
  
  // Update active sessions gauge
  activeSessionsGauge.set(activeSessions);
  
  res.json({
    status: 'ok',
    version: process.env.npm_package_version || '1.0.0',
    uptime: Math.floor((Date.now() - startTime) / 1000),
    sessions: activeSessions,
    timestamp: new Date().toISOString(),
  });
});

// Prometheus metrics endpoint
app.get('/metrics', async (req: Request, res: Response) => {
  try {
    res.set('Content-Type', register.contentType);
    res.send(await register.metrics());
  } catch (err) {
    logger.error('Failed to generate metrics:', err);
    res.status(500).send('Failed to generate metrics');
  }
});

// API routes
app.use('/api/sessions', sessionsRouter);
app.use('/api/sessions', messagesRouter);
app.use('/api/sessions', filesRouter);
app.use('/api/sessions', gitRouter);
app.use('/api/sessions', contextRouter);

// 404 handler
app.use((req: Request, res: Response) => {
  res.status(404).json({ error: 'Not found' });
});

// Error handler
app.use((err: Error, req: Request, res: Response, next: NextFunction) => {
  logger.error('Express error:', err);
  res.status(500).json({ error: 'Internal server error' });
});

// Create HTTP server
const server = createServer(app);

// Create WebSocket server
const wss = new WebSocketServer({ 
  server,
  path: '/ws',
});

// Initialize WebSocket handler
const wsHandler = new WebSocketHandler(wss);

// Start server
server.listen(config.port, config.host, () => {
  logger.info(`Bridge server started`);
  logger.info(`HTTP server listening on ${config.host}:${config.port}`);
  logger.info(`WebSocket server listening on ws://${config.host}:${config.port}/ws`);
  logger.info(`Max sessions: ${config.maxSessions}`);
  logger.info(`Session timeout: ${config.sessionTimeoutMs}ms`);
  logger.info(`Claude Code path: ${config.claudeCodePath}`);
});

// Graceful shutdown
const shutdown = async () => {
  logger.info('Shutting down bridge server...');

  // Stop accepting new connections
  server.close(() => {
    logger.info('HTTP server closed');
  });

  // Close WebSocket server
  wsHandler.close();
  logger.info('WebSocket server closed');

  // Kill all running CLI processes
  spawner.killAll();
  logger.info('All CLI processes terminated');

  // Close database
  db.close();
  logger.info('Database closed');

  process.exit(0);
};

// Handle shutdown signals
process.on('SIGTERM', shutdown);
process.on('SIGINT', shutdown);

// Handle uncaught errors
process.on('uncaughtException', (error: Error) => {
  logger.error('Uncaught exception:', error);
  shutdown();
});

process.on('unhandledRejection', (reason: any) => {
  logger.error('Unhandled rejection:', reason);
  shutdown();
});

export { app, server, wss };
export default app;
