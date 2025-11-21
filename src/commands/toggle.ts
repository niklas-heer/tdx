import chalk from 'chalk';
import { parseMarkdown } from '../todos/parser';
import { writeMarkdown } from '../todos/writer';
import { readTodoFile, writeTodoFile, ensureTodoFileExists } from '../fs/fileStore';

export function toggle(indexStr: string): void {
  try {
    const index = parseInt(indexStr, 10);

    if (isNaN(index) || index < 1) {
      console.error(chalk.red('Error:'), 'Todo index must be a positive number');
      process.exit(1);
    }

    ensureTodoFileExists();
    const content = readTodoFile();
    const model = parseMarkdown(content);

    if (index > model.todos.length) {
      console.error(chalk.red('Error:'), `Todo index ${index} out of range (1-${model.todos.length})`);
      process.exit(1);
    }

    const todo = model.todos[index - 1];
    todo.checked = !todo.checked;

    const updatedContent = writeMarkdown(model);
    writeTodoFile(updatedContent);

    const checkbox = todo.checked ? chalk.magenta('[✓]') : chalk.dim('[ ]');
    console.log(chalk.green('✓'), `Toggled: ${checkbox} ${todo.text}`);
  } catch (error) {
    console.error(chalk.red('Error:'), (error as Error).message);
    process.exit(1);
  }
}
