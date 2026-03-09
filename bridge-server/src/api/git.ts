/**
 * Git Operations API Endpoints
 */

import { Router, Request, Response } from 'express';
import { exec } from 'child_process';
import { promisify } from 'util';
import { db } from '../database.js';
import { logger } from '../logger.js';
import type { GitStatus, GitCommitRequest } from '../types/index.js';

const execAsync = promisify(exec);
const router = Router();

/**
 * Execute git command in project directory
 */
async function runGitCommand(projectPath: string, command: string): Promise<string> {
  try {
    const { stdout } = await execAsync(command, { cwd: projectPath });
    return stdout.trim();
  } catch (error: any) {
    logger.error(`Git command error: ${command}`, error);
    throw new Error(error.stderr || error.message);
  }
}

/**
 * GET /api/sessions/:id/git/status - Get git status
 */
router.get('/:id/git/status', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;

    // Check if session exists
    const session = db.getSession(id);
    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    // Get current branch
    const branch = await runGitCommand(session.projectPath, 'git branch --show-current');

    // Get ahead/behind status
    let ahead = 0;
    let behind = 0;
    try {
      const revList = await runGitCommand(
        session.projectPath,
        `git rev-list --left-right --count HEAD...@{u}`
      );
      const [aheadStr, behindStr] = revList.split('\t');
      ahead = parseInt(aheadStr, 10) || 0;
      behind = parseInt(behindStr, 10) || 0;
    } catch {
      // No upstream branch
    }

    // Get staged files
    const stagedOutput = await runGitCommand(session.projectPath, 'git diff --cached --name-only');
    const staged = stagedOutput ? stagedOutput.split('\n') : [];

    // Get unstaged files
    const unstagedOutput = await runGitCommand(session.projectPath, 'git diff --name-only');
    const unstaged = unstagedOutput ? unstagedOutput.split('\n') : [];

    // Get untracked files
    const untrackedOutput = await runGitCommand(session.projectPath, 'git ls-files --others --exclude-standard');
    const untracked = untrackedOutput ? untrackedOutput.split('\n') : [];

    const status: GitStatus = {
      branch,
      ahead,
      behind,
      staged: staged.filter(Boolean),
      unstaged: unstaged.filter(Boolean),
      untracked: untracked.filter(Boolean),
    };

    res.json({ status });
  } catch (error) {
    logger.error(`Error getting git status for session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Git command failed' });
  }
});

/**
 * POST /api/sessions/:id/git/commit - Commit changes
 */
router.post('/:id/git/commit', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const { message, files }: GitCommitRequest = req.body;

    if (!message) {
      return res.status(400).json({ error: 'Commit message required' });
    }

    // Check if session exists
    const session = db.getSession(id);
    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    // Stage files (if specified) or all changes
    if (files && files.length > 0) {
      for (const file of files) {
        await runGitCommand(session.projectPath, `git add "${file}"`);
      }
    } else {
      await runGitCommand(session.projectPath, 'git add -A');
    }

    // Commit
    const output = await runGitCommand(session.projectPath, `git commit -m "${message.replace(/"/g, '\\"')}"`);
    
    logger.info(`Committed changes in session ${id}: ${message}`);
    res.json({ message: 'Changes committed successfully', output });
  } catch (error) {
    logger.error(`Error committing changes for session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Git commit failed' });
  }
});

/**
 * POST /api/sessions/:id/git/push - Push to remote
 */
router.post('/:id/git/push', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;

    // Check if session exists
    const session = db.getSession(id);
    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    // Push changes
    const output = await runGitCommand(session.projectPath, 'git push');
    
    logger.info(`Pushed changes for session ${id}`);
    res.json({ message: 'Changes pushed successfully', output });
  } catch (error) {
    logger.error(`Error pushing changes for session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Git push failed' });
  }
});

export default router;
