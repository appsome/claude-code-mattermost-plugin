/**
 * Database Management
 */

import Database from 'better-sqlite3';
import fs from 'fs';
import path from 'path';
import { config } from './config.js';
import type { Session, Message, SessionStatus, MessageRole } from './types/index.js';

export class DatabaseManager {
  private db: Database.Database;

  constructor() {
    // Ensure database directory exists
    const dbDir = path.dirname(config.databasePath);
    if (!fs.existsSync(dbDir)) {
      fs.mkdirSync(dbDir, { recursive: true });
    }

    this.db = new Database(config.databasePath);
    this.initializeSchema();
  }

  private initializeSchema(): void {
    // Sessions table
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS sessions (
        id TEXT PRIMARY KEY,
        project_path TEXT NOT NULL,
        mattermost_user_id TEXT NOT NULL,
        mattermost_channel_id TEXT NOT NULL,
        cli_pid INTEGER,
        status TEXT CHECK(status IN ('active', 'stopped', 'error')) NOT NULL,
        created_at INTEGER NOT NULL,
        updated_at INTEGER NOT NULL
      );

      CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(mattermost_user_id);
      CREATE INDEX IF NOT EXISTS idx_sessions_channel ON sessions(mattermost_channel_id);
      CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
    `);

    // Messages table
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS messages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        session_id TEXT NOT NULL,
        role TEXT CHECK(role IN ('user', 'assistant', 'system')) NOT NULL,
        content TEXT NOT NULL,
        timestamp INTEGER NOT NULL,
        FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE
      );

      CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);
      CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
    `);
  }

  // Session operations
  createSession(session: Session): void {
    const stmt = this.db.prepare(`
      INSERT INTO sessions (id, project_path, mattermost_user_id, mattermost_channel_id, cli_pid, status, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `);
    
    stmt.run(
      session.id,
      session.projectPath,
      session.mattermostUserId,
      session.mattermostChannelId,
      session.cliPid,
      session.status,
      session.createdAt,
      session.updatedAt
    );
  }

  getSession(sessionId: string): Session | undefined {
    const stmt = this.db.prepare('SELECT * FROM sessions WHERE id = ?');
    const row = stmt.get(sessionId) as any;
    
    if (!row) return undefined;
    
    return {
      id: row.id,
      projectPath: row.project_path,
      mattermostUserId: row.mattermost_user_id,
      mattermostChannelId: row.mattermost_channel_id,
      cliPid: row.cli_pid,
      status: row.status as SessionStatus,
      createdAt: row.created_at,
      updatedAt: row.updated_at,
    };
  }

  getAllSessions(): Session[] {
    const stmt = this.db.prepare('SELECT * FROM sessions ORDER BY updated_at DESC');
    const rows = stmt.all() as any[];
    
    return rows.map(row => ({
      id: row.id,
      projectPath: row.project_path,
      mattermostUserId: row.mattermost_user_id,
      mattermostChannelId: row.mattermost_channel_id,
      cliPid: row.cli_pid,
      status: row.status as SessionStatus,
      createdAt: row.created_at,
      updatedAt: row.updated_at,
    }));
  }

  getSessionsByUser(userId: string): Session[] {
    const stmt = this.db.prepare('SELECT * FROM sessions WHERE mattermost_user_id = ? ORDER BY updated_at DESC');
    const rows = stmt.all(userId) as any[];
    
    return rows.map(row => ({
      id: row.id,
      projectPath: row.project_path,
      mattermostUserId: row.mattermost_user_id,
      mattermostChannelId: row.mattermost_channel_id,
      cliPid: row.cli_pid,
      status: row.status as SessionStatus,
      createdAt: row.created_at,
      updatedAt: row.updated_at,
    }));
  }

  updateSessionStatus(sessionId: string, status: SessionStatus, cliPid?: number): void {
    const now = Date.now();
    
    if (cliPid !== undefined) {
      const stmt = this.db.prepare('UPDATE sessions SET status = ?, cli_pid = ?, updated_at = ? WHERE id = ?');
      stmt.run(status, cliPid, now, sessionId);
    } else {
      const stmt = this.db.prepare('UPDATE sessions SET status = ?, updated_at = ? WHERE id = ?');
      stmt.run(status, now, sessionId);
    }
  }

  deleteSession(sessionId: string): void {
    const stmt = this.db.prepare('DELETE FROM sessions WHERE id = ?');
    stmt.run(sessionId);
  }

  // Message operations
  addMessage(message: Omit<Message, 'id'>): Message {
    const stmt = this.db.prepare(`
      INSERT INTO messages (session_id, role, content, timestamp)
      VALUES (?, ?, ?, ?)
    `);
    
    const result = stmt.run(
      message.sessionId,
      message.role,
      message.content,
      message.timestamp
    );
    
    return {
      id: result.lastInsertRowid as number,
      ...message,
    };
  }

  getMessages(sessionId: string, limit?: number): Message[] {
    let stmt;
    let rows;
    
    if (limit) {
      stmt = this.db.prepare(`
        SELECT * FROM messages 
        WHERE session_id = ? 
        ORDER BY timestamp DESC 
        LIMIT ?
      `);
      rows = stmt.all(sessionId, limit) as any[];
      rows.reverse(); // Return in chronological order
    } else {
      stmt = this.db.prepare(`
        SELECT * FROM messages 
        WHERE session_id = ? 
        ORDER BY timestamp ASC
      `);
      rows = stmt.all(sessionId) as any[];
    }
    
    return rows.map(row => ({
      id: row.id,
      sessionId: row.session_id,
      role: row.role as MessageRole,
      content: row.content,
      timestamp: row.timestamp,
    }));
  }

  close(): void {
    this.db.close();
  }
}

export const db = new DatabaseManager();
export default db;
