/**
 * Messages API Endpoints
 */

import { Router, Request, Response } from 'express';
import { db } from '../database.js';
import { spawner } from '../spawner.js';
import { logger } from '../logger.js';
import type { SendMessageRequest } from '../types/index.js';

const router = Router();

/**
 * POST /api/sessions/:id/message - Send message to session
 */
router.post('/:id/message', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const { message }: SendMessageRequest = req.body;

    if (!message) {
      return res.status(400).json({ error: 'Missing required field: message' });
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

    // Save user message
    const now = Date.now();
    db.addMessage({
      sessionId: id,
      role: 'user',
      content: message,
      timestamp: now,
    });

    // Send message to CLI process
    const success = spawner.sendInput(id, message);
    if (!success) {
      return res.status(500).json({ error: 'Failed to send message to CLI process' });
    }

    logger.info(`Sent message to session ${id}`);
    res.json({ message: 'Message sent successfully' });
  } catch (error) {
    logger.error(`Error sending message to session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * GET /api/sessions/:id/messages - Get message history
 */
router.get('/:id/messages', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const { limit } = req.query;

    // Check if session exists
    const session = db.getSession(id);
    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    // Get messages
    const messages = db.getMessages(id, limit ? parseInt(limit as string, 10) : undefined);

    res.json({ messages });
  } catch (error) {
    logger.error(`Error getting messages for session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

export default router;
