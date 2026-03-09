/**
 * Bridge Server Types
 */

export interface Session {
  id: string;
  projectPath: string;
  mattermostUserId: string;
  mattermostChannelId: string;
  cliPid: number | null;
  status: SessionStatus;
  createdAt: number;
  updatedAt: number;
}

export type SessionStatus = 'active' | 'stopped' | 'error';

export interface Message {
  id: number;
  sessionId: string;
  role: MessageRole;
  content: string;
  timestamp: number;
}

export type MessageRole = 'user' | 'assistant' | 'system';

export interface FileNode {
  name: string;
  path: string;
  type: 'file' | 'directory';
  size?: number;
  children?: FileNode[];
}

export interface GitStatus {
  branch: string;
  ahead: number;
  behind: number;
  staged: string[];
  unstaged: string[];
  untracked: string[];
}

export interface CreateSessionRequest {
  projectPath: string;
  mattermostUserId: string;
  mattermostChannelId: string;
}

export interface SendMessageRequest {
  message: string;
}

export interface FileOperationRequest {
  path: string;
  content?: string;
}

export interface GitCommitRequest {
  message: string;
  files?: string[];
}

export interface WebSocketMessage {
  type: 'output' | 'status' | 'error' | 'file_change';
  sessionId: string;
  data: any;
  timestamp: number;
}

export interface CLIProcess {
  pid: number;
  sessionId: string;
  process: any; // Child process
  startTime: number;
}

export interface Config {
  port: number;
  host: string;
  databasePath: string;
  claudeCodePath: string;
  cursorPath: string;
  maxSessions: number;
  sessionTimeoutMs: number;
  logLevel: string;
  logFile: string;
  corsOrigin: string;
}
