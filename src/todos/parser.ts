import { FileModel, Todo, isTodoLine, parseTodoLine } from './model.ts';

/**
 * Parse a markdown file content into a FileModel
 */
export function parseMarkdown(content: string): FileModel {
  const lines = content.split('\n');
  const todos: Todo[] = [];
  let todoIndex = 0;

  for (let lineNo = 0; lineNo < lines.length; lineNo++) {
    const line = lines[lineNo];
    if (isTodoLine(line)) {
      const todo = parseTodoLine(line, lineNo, todoIndex + 1);
      if (todo) {
        todos.push(todo);
        todoIndex++;
      }
    }
  }

  return { lines, todos };
}

/**
 * Parse markdown file from a string with proper newline handling
 */
export function parseFile(content: string): FileModel {
  return parseMarkdown(content);
}
