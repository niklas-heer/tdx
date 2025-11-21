import fs from 'fs';
import path from 'path';

const TODO_FILE = 'todo.md';
const DEFAULT_HEADER = '# Todos\n';

/**
 * Gets the path to todo.md in the current working directory
 */
export function getTodoFilePath(): string {
  return path.join(process.cwd(), TODO_FILE);
}

/**
 * Checks if todo.md exists
 */
export function todoFileExists(): boolean {
  try {
    fs.accessSync(getTodoFilePath(), fs.constants.F_OK);
    return true;
  } catch {
    return false;
  }
}

/**
 * Creates todo.md with default header if it doesn't exist
 */
export function ensureTodoFileExists(): void {
  const filePath = getTodoFilePath();
  if (!todoFileExists()) {
    fs.writeFileSync(filePath, DEFAULT_HEADER);
  }
}

/**
 * Reads todo.md content synchronously
 */
export function readTodoFile(): string {
  const filePath = getTodoFilePath();
  try {
    return fs.readFileSync(filePath, 'utf-8');
  } catch (error) {
    throw new Error(`Failed to read ${TODO_FILE}: ${(error as Error).message}`);
  }
}

/**
 * Writes content to todo.md atomically
 */
export function writeTodoFile(content: string): void {
  const filePath = getTodoFilePath();
  const tempPath = `${filePath}.tmp`;

  try {
    // Write to temporary file
    fs.writeFileSync(tempPath, content);

    // Atomic rename
    fs.renameSync(tempPath, filePath);
  } catch (error) {
    // Clean up temp file if it exists
    try {
      fs.unlinkSync(tempPath);
    } catch {
      // Ignore cleanup errors
    }
    throw new Error(`Failed to write ${TODO_FILE}: ${(error as Error).message}`);
  }
}
