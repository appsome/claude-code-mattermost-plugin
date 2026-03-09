/**
 * Configuration Management
 */

import dotenv from 'dotenv';
import path from 'path';
import { fileURLToPath } from 'url';
import type { Config } from './types/index.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Load environment variables
dotenv.config({ path: path.join(__dirname, '../.env') });

export const config: Config = {
  port: parseInt(process.env.PORT || '3002', 10),
  host: process.env.HOST || '0.0.0.0',
  databasePath: process.env.DATABASE_PATH || path.join(__dirname, '../data/sessions.db'),
  claudeCodePath: process.env.CLAUDE_CODE_PATH || '/usr/local/bin/claude-code',
  cursorPath: process.env.CURSOR_PATH || '/usr/local/bin/cursor',
  maxSessions: parseInt(process.env.MAX_SESSIONS || '10', 10),
  sessionTimeoutMs: parseInt(process.env.SESSION_TIMEOUT_MS || '3600000', 10),
  logLevel: process.env.LOG_LEVEL || 'info',
  logFile: process.env.LOG_FILE || path.join(__dirname, '../logs/bridge-server.log'),
  corsOrigin: process.env.CORS_ORIGIN || '*',
};

export default config;
