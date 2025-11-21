import fs from "fs";
import path from "path";

const DEFAULT_TODO_FILE = "todo.md";
const DEFAULT_HEADER = "# Todos\n";

// Global custom path - can be set before operations
let customTodoFilePath: string | null = null;

/**
 * Sets a custom path for the todo file
 */
export function setTodoFilePath(filePath: string): void {
  // Resolve to absolute path
  customTodoFilePath = path.isAbsolute(filePath)
    ? filePath
    : path.resolve(process.cwd(), filePath);
}

/**
 * Gets the path to the todo file
 */
export function getTodoFilePath(): string {
  if (customTodoFilePath) {
    return customTodoFilePath;
  }
  return path.join(process.cwd(), DEFAULT_TODO_FILE);
}

/**
 * Checks if the todo file exists
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
 * Creates the todo file with default header if it doesn't exist
 */
export function ensureTodoFileExists(): void {
  const filePath = getTodoFilePath();
  if (!todoFileExists()) {
    fs.writeFileSync(filePath, DEFAULT_HEADER);
  }
}

/**
 * Reads todo file content synchronously
 */
export function readTodoFile(): string {
  const filePath = getTodoFilePath();
  try {
    return fs.readFileSync(filePath, "utf-8");
  } catch (error) {
    throw new Error(`Failed to read ${filePath}: ${(error as Error).message}`);
  }
}

/**
 * Writes content to todo file atomically
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
    throw new Error(`Failed to write ${filePath}: ${(error as Error).message}`);
  }
}
