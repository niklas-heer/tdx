import { FileModel, formatTodoLine } from "./model";

/**
 * Writes a FileModel back to markdown format
 * Replaces only todo lines, preserves all other content exactly
 */
export function writeMarkdown(model: FileModel): string {
  const result: string[] = [];

  for (let i = 0; i < model.lines.length; i++) {
    const line = model.lines[i];

    // Find if this line corresponds to a todo
    const todo = model.todos.find((t) => t.lineNo === i);

    if (todo) {
      // This is a todo line, write the updated version
      result.push(formatTodoLine(todo));
    } else {
      // Non-todo line, preserve exactly
      result.push(line);
    }
  }

  // Clean up markdown: remove all blank lines between todo items
  const cleaned = result
    .join("\n")
    .replace(/\n{2,}/g, "\n") // Replace 2+ newlines with 1
    .replace(/^\n+/, "") // Remove leading newlines
    .replace(/\n*$/, "\n"); // Ensure single trailing newline

  return cleaned;
}

/**
 * Synchronous atomic write to file using Bun
 */
export function writeAtomicallySyncronously(
  filePath: string,
  content: string,
): void {
  const fs = require("fs");
  const path = require("path");

  const dir = path.dirname(filePath);
  const basename = path.basename(filePath);
  const tempPath = path.join(dir, `.${basename}.tmp.${Date.now()}`);

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
    throw error;
  }
}
