/**
 * Session API Endpoints
 */

import { Router, Request, Response } from 'express';
import { v4 as uuidv4 } from 'uuid';
import { db } from '../database.js';
import { spawner } from '../spawner.js';
import { logger } from '../logger.js';
import type { CreateSessionRequest, Session } from '../types/index.js';

const router = Router();

/**
 * POST /api/sessions - Create new session
 */
router.post('/', async (req: Request, res: Response) => {
  try {
    const { projectPath, mattermostUserId, mattermostChannelId }: CreateSessionRequest = req.body;

    if (!projectPath || !mattermostUserId || !mattermostChannelId) {
      return res.status(400).json({
        error: 'Missing required fields: projectPath, mattermostUserId, mattermostChannelId',
      });
    }

    // Create session
    const sessionId = uuidv4();
    const now = Date.now();
    
    const session: Session = {
      id: sessionId,
      projectPath,
      mattermostUserId,
      mattermostChannelId,
      cliPid: null,
      status: 'active',
      createdAt: now,
      updatedAt: now,
    };

    // Save to database
    db.createSession(session);
    logger.info(`Created session ${sessionId} for user ${mattermostUserId}`);

    // Spawn CLI process
    try {
      const cliProcess = spawner.spawn(sessionId, projectPath);
      db.updateSessionStatus(sessionId, 'active', cliProcess.pid);
      session.cliPid = cliProcess.pid;
    } catch (error) {
      logger.error(`Failed to spawn CLI for session ${sessionId}:`, error);
      db.updateSessionStatus(sessionId, 'error');
      session.status = 'error';
    }

    res.status(201).json({ session });
  } catch (error) {
    logger.error('Error creating session:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * GET /api/sessions - List all sessions
 */
router.get('/', async (req: Request, res: Response) => {
  try {
    const { userId } = req.query;

    let sessions: Session[];
    if (userId) {
      sessions = db.getSessionsByUser(userId as string);
    } else {
      sessions = db.getAllSessions();
    }

    res.json({ sessions });
  } catch (error) {
    logger.error('Error listing sessions:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * GET /api/sessions/:id - Get session details
 */
router.get('/:id', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const session = db.getSession(id);

    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    // Check if process is still running
    const isRunning = spawner.isRunning(id);
    if (!isRunning && session.status === 'active') {
      db.updateSessionStatus(id, 'stopped');
      session.status = 'stopped';
    }

    res.json({ session });
  } catch (error) {
    logger.error(`Error getting session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * DELETE /api/sessions/:id - Stop and delete session
 */
router.delete('/:id', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const session = db.getSession(id);

    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    // Kill CLI process if running
    if (spawner.isRunning(id)) {
      spawner.kill(id);
      logger.info(`Killed CLI process for session ${id}`);
    }

    // Delete from database
    db.deleteSession(id);
    logger.info(`Deleted session ${id}`);

    res.json({ message: 'Session deleted successfully' });
  } catch (error) {
    logger.error(`Error deleting session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

export default router;
