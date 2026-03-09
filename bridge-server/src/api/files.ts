/**
 * File Operations API Endpoints
 */

import { Router, Request, Response } from 'express';
import fs from 'fs/promises';
import path from 'path';
import { db } from '../database.js';
import { logger } from '../logger.js';
import type { FileNode, FileOperationRequest } from '../types/index.js';

const router = Router();

/**
 * Validate file path to prevent directory traversal
 */
function validatePath(sessionProjectPath: string, requestedPath: string): string | null {
  const resolved = path.resolve(sessionProjectPath, requestedPath);
  if (!resolved.startsWith(sessionProjectPath)) {
    return null;
  }
  return resolved;
}

/**
 * Build file tree recursively
 */
async function buildFileTree(dirPath: string, maxDepth: number = 3, currentDepth: number = 0): Promise<FileNode[]> {
  if (currentDepth >= maxDepth) {
    return [];
  }

  const entries = await fs.readdir(dirPath, { withFileTypes: true });
  const nodes: FileNode[] = [];

  for (const entry of entries) {
    // Skip hidden files and common ignore patterns
    if (entry.name.startsWith('.') || entry.name === 'node_modules') {
      continue;
    }

    const fullPath = path.join(dirPath, entry.name);
    const node: FileNode = {
      name: entry.name,
      path: fullPath,
      type: entry.isDirectory() ? 'directory' : 'file',
    };

    if (entry.isFile()) {
      try {
        const stats = await fs.stat(fullPath);
        node.size = stats.size;
      } catch (error) {
        logger.warn(`Error getting file stats for ${fullPath}:`, error);
      }
    }

    if (entry.isDirectory()) {
      node.children = await buildFileTree(fullPath, maxDepth, currentDepth + 1);
    }

    nodes.push(node);
  }

  return nodes.sort((a, b) => {
    // Directories first, then alphabetical
    if (a.type !== b.type) {
      return a.type === 'directory' ? -1 : 1;
    }
    return a.name.localeCompare(b.name);
  });
}

/**
 * GET /api/sessions/:id/files - List project files
 */
router.get('/:id/files', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const { path: requestedPath } = req.query;

    // Check if session exists
    const session = db.getSession(id);
    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    const targetPath = requestedPath 
      ? validatePath(session.projectPath, requestedPath as string)
      : session.projectPath;

    if (!targetPath) {
      return res.status(400).json({ error: 'Invalid path' });
    }

    const files = await buildFileTree(targetPath);
    res.json({ files });
  } catch (error) {
    logger.error(`Error listing files for session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * GET /api/sessions/:id/files/:path(*) - Get file content
 */
router.get('/:id/files/*', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const requestedPath = req.params[0];

    if (!requestedPath) {
      return res.status(400).json({ error: 'File path required' });
    }

    // Check if session exists
    const session = db.getSession(id);
    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    const filePath = validatePath(session.projectPath, requestedPath);
    if (!filePath) {
      return res.status(400).json({ error: 'Invalid path' });
    }

    // Check if file exists
    try {
      const stats = await fs.stat(filePath);
      if (!stats.isFile()) {
        return res.status(400).json({ error: 'Path is not a file' });
      }
    } catch (error) {
      return res.status(404).json({ error: 'File not found' });
    }

    // Read file content
    const content = await fs.readFile(filePath, 'utf-8');
    res.json({ path: requestedPath, content });
  } catch (error) {
    logger.error(`Error reading file for session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * PUT /api/sessions/:id/files/:path(*) - Update file
 */
router.put('/:id/files/*', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const requestedPath = req.params[0];
    const { content }: FileOperationRequest = req.body;

    if (!requestedPath || content === undefined) {
      return res.status(400).json({ error: 'File path and content required' });
    }

    // Check if session exists
    const session = db.getSession(id);
    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    const filePath = validatePath(session.projectPath, requestedPath);
    if (!filePath) {
      return res.status(400).json({ error: 'Invalid path' });
    }

    // Write file
    await fs.writeFile(filePath, content, 'utf-8');
    logger.info(`Updated file ${requestedPath} in session ${id}`);

    res.json({ message: 'File updated successfully', path: requestedPath });
  } catch (error) {
    logger.error(`Error updating file for session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * POST /api/sessions/:id/files - Create file
 */
router.post('/:id/files', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const { path: requestedPath, content }: FileOperationRequest = req.body;

    if (!requestedPath) {
      return res.status(400).json({ error: 'File path required' });
    }

    // Check if session exists
    const session = db.getSession(id);
    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    const filePath = validatePath(session.projectPath, requestedPath);
    if (!filePath) {
      return res.status(400).json({ error: 'Invalid path' });
    }

    // Check if file already exists
    try {
      await fs.access(filePath);
      return res.status(409).json({ error: 'File already exists' });
    } catch {
      // File doesn't exist, continue
    }

    // Ensure directory exists
    await fs.mkdir(path.dirname(filePath), { recursive: true });

    // Create file
    await fs.writeFile(filePath, content || '', 'utf-8');
    logger.info(`Created file ${requestedPath} in session ${id}`);

    res.status(201).json({ message: 'File created successfully', path: requestedPath });
  } catch (error) {
    logger.error(`Error creating file for session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

/**
 * DELETE /api/sessions/:id/files/:path(*) - Delete file
 */
router.delete('/:id/files/*', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const requestedPath = req.params[0];

    if (!requestedPath) {
      return res.status(400).json({ error: 'File path required' });
    }

    // Check if session exists
    const session = db.getSession(id);
    if (!session) {
      return res.status(404).json({ error: 'Session not found' });
    }

    const filePath = validatePath(session.projectPath, requestedPath);
    if (!filePath) {
      return res.status(400).json({ error: 'Invalid path' });
    }

    // Delete file
    await fs.unlink(filePath);
    logger.info(`Deleted file ${requestedPath} in session ${id}`);

    res.json({ message: 'File deleted successfully', path: requestedPath });
  } catch (error) {
    logger.error(`Error deleting file for session ${req.params.id}:`, error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

export default router;
