export interface Todo {
  index: number;        // 1-based index among todos
  checked: boolean;
  text: string;
  lineNo: number;       // 0-based line number in file
}

export interface FileModel {
  lines: string[];      // All original lines from file
  todos: Todo[];        // Parsed todos
}

/**
 * Check if a line is a todo line (starts with `- [ ] ` or `- [x] `)
 */
export function isTodoLine(line: string): boolean {
  return line.startsWith('- [ ] ') || line.startsWith('- [x] ');
}

/**
 * Parse a single todo line into Todo object
 */
export function parseTodoLine(line: string, lineNo: number, todoIndex: number): Todo | null {
  if (line.startsWith('- [ ] ')) {
    return {
      index: todoIndex,
      checked: false,
      text: line.slice(6), // Remove '- [ ] ' prefix
      lineNo,
    };
  }
  if (line.startsWith('- [x] ')) {
    return {
      index: todoIndex,
      checked: true,
      text: line.slice(6), // Remove '- [x] ' prefix
      lineNo,
    };
  }
  return null;
}

/**
 * Format a Todo back into a markdown line
 */
export function formatTodoLine(todo: Todo): string {
  const checkbox = todo.checked ? '[x]' : '[ ]';
  return `- ${checkbox} ${todo.text}`;
}
