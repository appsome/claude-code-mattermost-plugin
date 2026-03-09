/**
 * CLI Process Spawner
 * Manages Claude Code CLI processes for each session
 */

import { spawn, ChildProcess } from 'child_process';
import { EventEmitter } from 'events';
import { config } from './config.js';
import { logger } from './logger.js';
import type { CLIProcess } from './types/index.js';

export interface SpawnerEvents {
  output: (sessionId: string, data: string) => void;
  error: (sessionId: string, error: string) => void;
  exit: (sessionId: string, code: number | null) => void;
}

export class CLISpawner extends EventEmitter {
  private processes: Map<string, CLIProcess> = new Map();

  /**
   * Spawn a new Claude Code CLI process
   */
  spawn(sessionId: string, projectPath: string): CLIProcess {
    // Check if session already has a running process
    const existing = this.processes.get(sessionId);
    if (existing) {
      logger.warn(`Session ${sessionId} already has a running process`);
      return existing;
    }

    // Spawn Claude Code CLI
    const cliPath = config.claudeCodePath;
    const args = [projectPath]; // Claude Code takes project path as argument
    
    logger.info(`Spawning Claude Code CLI for session ${sessionId}: ${cliPath} ${args.join(' ')}`);
    
    const child = spawn(cliPath, args, {
      cwd: projectPath,
      env: {
        ...process.env,
        CLAUDE_CODE_INTERACTIVE: 'false', // Non-interactive mode
        CLAUDE_CODE_OUTPUT: 'json', // JSON output format
      },
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    const cliProcess: CLIProcess = {
      pid: child.pid!,
      sessionId,
      process: child,
      startTime: Date.now(),
    };

    // Handle stdout
    child.stdout?.on('data', (data: Buffer) => {
      const output = data.toString();
      logger.debug(`[${sessionId}] stdout: ${output}`);
      this.emit('output', sessionId, output);
    });

    // Handle stderr
    child.stderr?.on('data', (data: Buffer) => {
      const error = data.toString();
      logger.debug(`[${sessionId}] stderr: ${error}`);
      this.emit('error', sessionId, error);
    });

    // Handle process exit
    child.on('exit', (code: number | null, signal: NodeJS.Signals | null) => {
      logger.info(`[${sessionId}] Process exited with code ${code}, signal ${signal}`);
      this.processes.delete(sessionId);
      this.emit('exit', sessionId, code);
    });

    // Handle process errors
    child.on('error', (error: Error) => {
      logger.error(`[${sessionId}] Process error:`, error);
      this.emit('error', sessionId, error.message);
      this.processes.delete(sessionId);
    });

    this.processes.set(sessionId, cliProcess);
    return cliProcess;
  }

  /**
   * Send input to a CLI process
   */
  sendInput(sessionId: string, input: string): boolean {
    const cliProcess = this.processes.get(sessionId);
    if (!cliProcess || !cliProcess.process.stdin) {
      logger.warn(`Cannot send input to session ${sessionId}: process not found or stdin unavailable`);
      return false;
    }

    try {
      cliProcess.process.stdin.write(input + '\n');
      return true;
    } catch (error) {
      logger.error(`Error sending input to session ${sessionId}:`, error);
      return false;
    }
  }

  /**
   * Kill a CLI process
   */
  kill(sessionId: string, signal: NodeJS.Signals = 'SIGTERM'): boolean {
    const cliProcess = this.processes.get(sessionId);
    if (!cliProcess) {
      logger.warn(`Cannot kill session ${sessionId}: process not found`);
      return false;
    }

    try {
      cliProcess.process.kill(signal);
      this.processes.delete(sessionId);
      logger.info(`Killed CLI process for session ${sessionId}`);
      return true;
    } catch (error) {
      logger.error(`Error killing process for session ${sessionId}:`, error);
      return false;
    }
  }

  /**
   * Check if a session has a running process
   */
  isRunning(sessionId: string): boolean {
    return this.processes.has(sessionId);
  }

  /**
   * Get process for a session
   */
  getProcess(sessionId: string): CLIProcess | undefined {
    return this.processes.get(sessionId);
  }

  /**
   * Get all running processes
   */
  getAllProcesses(): CLIProcess[] {
    return Array.from(this.processes.values());
  }

  /**
   * Kill all running processes
   */
  killAll(): void {
    logger.info(`Killing all ${this.processes.size} CLI processes`);
    for (const [sessionId] of this.processes) {
      this.kill(sessionId);
    }
  }
}

// Singleton instance
export const spawner = new CLISpawner();
export default spawner;
