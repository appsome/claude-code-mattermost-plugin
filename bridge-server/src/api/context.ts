/**
 * Context API Endpoints
 * Handles injection of external context (e.g., Mattermost threads) into sessions
 */

import { Router, Request, Response } from 'express';
import { db } from '../database.js';
import { spawner } from '../spawner.js';
import { logger } from '../logger.js';

const router = Router();

interface ContextRequest {
  source: string;
  threadId?: string;
  content: string;
  action?: string;
  metadata?: {
    channelName?: string;
    rootPostId?: string;
    messageCount?: number;
    participants?: string[];
  };
}

/**
 * POST /api/sessions/:id/context - Inject context into session
 */
router.post('/:id/context', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const contextReq: ContextRequest = req.body;

    if (!contextReq.content) {
      return res.status(400).json({ error: 'Missing required field: content' });
    }

    // Check if session exists
    const session = db.getSession(id);
    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    // Check if CLI process is running
    if (!spawner.isRunning(id)) {
      return res.status(400).json({ error: 'CLI process not running for this session' });
    }

    // Format context message
    const contextMessage = formatContextMessage(contextReq);

    // Save context as system message
    const now = Date.now();
    db.addMessage({
      sessionId: id,
      role: 'system',
      content: contextMessage,
      timestamp: now,
    });

    // Send context to CLI process
    const success = spawner.sendInput(id, contextMessage);
    if (!success) {
      return res.status(500).json({ error: 'Failed to send context to CLI process' });
    }

    logger.info(`Injected context into session ${id}`, {
      source: contextReq.source,
      contentLength: contextReq.content.length,
      action: contextReq.action,
    });

    // If action specified, send it as a follow-up user message
    if (contextReq.action) {
      const actionMessage = formatActionMessage(contextReq.action);
      
      // Save action as user message
      db.addMessage({
        sessionId: id,
        role: 'user',
        content: actionMessage,
        timestamp: now + 1,
      });

      // Send action to CLI
      spawner.sendInput(id, actionMessage);
      logger.info(`Sent action to session ${id}: ${contextReq.action}`);
    }

    res.json({
      message: 'Context injected successfully',
      metadata: {
        contentLength: contextReq.content.length,
        actionRequested: !!contextReq.action,
        source: contextReq.source,
      },
    });
  } catch (error) {
    logger.error(`Error injecting context into session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * Format context message for CLI
 */
function formatContextMessage(contextReq: ContextRequest): string {
  let message = '--- CONTEXT START ---\n\n';
  
  if (contextReq.source === 'mattermost-thread' && contextReq.metadata) {
    message += `Source: Mattermost Thread in #${contextReq.metadata.channelName || 'unknown'}\n`;
    message += `Messages: ${contextReq.metadata.messageCount || 'unknown'}\n`;
    if (contextReq.metadata.participants && contextReq.metadata.participants.length > 0) {
      message += `Participants: ${contextReq.metadata.participants.join(', ')}\n`;
    }
    message += '\n';
  }

  message += contextReq.content;
  message += '\n\n--- CONTEXT END ---';

  return message;
}

/**
 * Format action message for CLI
 */
function formatActionMessage(action: string): string {
  const actionMap: Record<string, string> = {
    summarize: 'Please summarize the above thread context.',
    sum: 'Please summarize the above thread context.',
    implement: 'Please implement the changes discussed in the above thread context.',
    impl: 'Please implement the changes discussed in the above thread context.',
    review: 'Please review the discussion in the above thread context and provide feedback.',
    fix: 'Please help fix the issues discussed in the above thread context.',
  };

  return actionMap[action.toLowerCase()] || action;
}

export default router;
