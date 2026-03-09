/**
 * WebSocket Handler
 * Manages real-time CLI output streaming
 */

import { WebSocketServer, WebSocket } from 'ws';
import { IncomingMessage } from 'http';
import { spawner } from '../spawner.js';
import { logger } from '../logger.js';
import type { WebSocketMessage } from '../types/index.js';

interface ClientConnection {
  ws: WebSocket;
  sessionId?: string;
  alive: boolean;
}

export class WebSocketHandler {
  private wss: WebSocketServer;
  private clients: Map<WebSocket, ClientConnection> = new Map();
  private heartbeatInterval: NodeJS.Timeout;

  constructor(wss: WebSocketServer) {
    this.wss = wss;
    this.setupEventListeners();
    this.heartbeatInterval = this.startHeartbeat();
  }

  private setupEventListeners(): void {
    // Handle new WebSocket connections
    this.wss.on('connection', (ws: WebSocket, req: IncomingMessage) => {
      const client: ClientConnection = {
        ws,
        alive: true,
      };
      
      this.clients.set(ws, client);
      logger.info(`WebSocket client connected (${this.clients.size} total)`);

      // Handle pong responses
      ws.on('pong', () => {
        const client = this.clients.get(ws);
        if (client) {
          client.alive = true;
        }
      });

      // Handle incoming messages
      ws.on('message', (data: Buffer) => {
        try {
          const message = JSON.parse(data.toString());
          this.handleClientMessage(ws, message);
        } catch (error) {
          logger.error('Error parsing WebSocket message:', error);
        }
      });

      // Handle client disconnect
      ws.on('close', () => {
        this.clients.delete(ws);
        logger.info(`WebSocket client disconnected (${this.clients.size} remaining)`);
      });

      // Handle errors
      ws.on('error', (error: Error) => {
        logger.error('WebSocket error:', error);
      });

      // Send welcome message
      this.sendMessage(ws, {
        type: 'status',
        sessionId: '',
        data: { message: 'Connected to bridge server' },
        timestamp: Date.now(),
      });
    });

    // Listen to spawner events
    spawner.on('output', (sessionId: string, data: string) => {
      this.broadcastToSession(sessionId, {
        type: 'output',
        sessionId,
        data: { output: data },
        timestamp: Date.now(),
      });
    });

    spawner.on('error', (sessionId: string, error: string) => {
      this.broadcastToSession(sessionId, {
        type: 'error',
        sessionId,
        data: { error },
        timestamp: Date.now(),
      });
    });

    spawner.on('exit', (sessionId: string, code: number | null) => {
      this.broadcastToSession(sessionId, {
        type: 'status',
        sessionId,
        data: { status: 'stopped', exitCode: code },
        timestamp: Date.now(),
      });
    });
  }

  private handleClientMessage(ws: WebSocket, message: any): void {
    const client = this.clients.get(ws);
    if (!client) return;

    // Handle subscription to session
    if (message.type === 'subscribe' && message.sessionId) {
      client.sessionId = message.sessionId;
      logger.info(`Client subscribed to session ${message.sessionId}`);
      
      this.sendMessage(ws, {
        type: 'status',
        sessionId: message.sessionId,
        data: { message: `Subscribed to session ${message.sessionId}` },
        timestamp: Date.now(),
      });
    }

    // Handle unsubscribe
    if (message.type === 'unsubscribe') {
      const previousSession = client.sessionId;
      client.sessionId = undefined;
      logger.info(`Client unsubscribed from session ${previousSession}`);
    }
  }

  private sendMessage(ws: WebSocket, message: WebSocketMessage): void {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message));
    }
  }

  private broadcastToSession(sessionId: string, message: WebSocketMessage): void {
    let sentCount = 0;
    
    for (const [ws, client] of this.clients) {
      if (client.sessionId === sessionId) {
        this.sendMessage(ws, message);
        sentCount++;
      }
    }

    if (sentCount > 0) {
      logger.debug(`Broadcast to ${sentCount} client(s) for session ${sessionId}`);
    }
  }

  private startHeartbeat(): NodeJS.Timeout {
    return setInterval(() => {
      for (const [ws, client] of this.clients) {
        if (!client.alive) {
          logger.info('Terminating inactive WebSocket client');
          ws.terminate();
          this.clients.delete(ws);
          continue;
        }

        client.alive = false;
        ws.ping();
      }
    }, 30000); // 30 seconds
  }

  public close(): void {
    clearInterval(this.heartbeatInterval);
    
    for (const [ws] of this.clients) {
      ws.close();
    }
    
    this.clients.clear();
    this.wss.close();
  }
}

export default WebSocketHandler;
