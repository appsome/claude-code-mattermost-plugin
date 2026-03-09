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

/**
 * POST /api/sessions/:id/approve - Approve a code change
 */
router.post('/:id/approve', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const { changeId } = req.body;

    if (!changeId) {
      return res.status(400).json({ error: 'Missing changeId' });
    }

    const session = db.getSession(id);
    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    if (!spawner.isRunning(id)) {
      return res.status(400).json({ error: 'Session CLI is not running' });
    }

    // Send approval signal to CLI
    // This would typically be sent via stdin or a special approval mechanism
    spawner.sendInput(id, `approve ${changeId}\n`);
    logger.info(`Approved change ${changeId} for session ${id}`);

    res.json({ success: true });
  } catch (error) {
    logger.error(`Error approving change for session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * POST /api/sessions/:id/reject - Reject a code change
 */
router.post('/:id/reject', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const { changeId } = req.body;

    if (!changeId) {
      return res.status(400).json({ error: 'Missing changeId' });
    }

    const session = db.getSession(id);
    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    if (!spawner.isRunning(id)) {
      return res.status(400).json({ error: 'Session CLI is not running' });
    }

    // Send rejection signal to CLI
    spawner.sendInput(id, `reject ${changeId}\n`);
    logger.info(`Rejected change ${changeId} for session ${id}`);

    res.json({ success: true });
  } catch (error) {
    logger.error(`Error rejecting change for session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * POST /api/sessions/:id/modify - Request modifications to a code change
 */
router.post('/:id/modify', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const { changeId, instructions } = req.body;

    if (!changeId || !instructions) {
      return res.status(400).json({ error: 'Missing changeId or instructions' });
    }

    const session = db.getSession(id);
    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    if (!spawner.isRunning(id)) {
      return res.status(400).json({ error: 'Session CLI is not running' });
    }

    // Send modification request to CLI
    spawner.sendInput(id, `modify ${changeId}: ${instructions}\n`);
    logger.info(`Requested modification for change ${changeId} in session ${id}`);

    res.json({ success: true });
  } catch (error) {
    logger.error(`Error modifying change for session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * POST /api/sessions/:id/file - Get file content from session's project
 */
router.post('/:id/file', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const { filename } = req.body;

    if (!filename) {
      return res.status(400).json({ error: 'Missing filename' });
    }

    const session = db.getSession(id);
    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    // Read file from project path
    const fs = await import('fs/promises');
    const path = await import('path');
    const filePath = path.join(session.projectPath, filename);

    try {
      const content = await fs.readFile(filePath, 'utf-8');
      res.json({ content });
    } catch (error) {
      logger.error(`Error reading file ${filePath}:`, error);
      res.status(404).json({ error: 'File not found' });
    }
  } catch (error) {
    logger.error(`Error getting file for session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

export default router;
