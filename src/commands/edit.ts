import chalk from 'chalk';
import { parseMarkdown } from '../todos/parser';
import { writeMarkdown } from '../todos/writer';
import { readTodoFile, writeTodoFile, ensureTodoFileExists } from '../fs/fileStore';

export function edit(indexStr: string, newText: string): void {
  try {
    const index = parseInt(indexStr, 10);

    if (isNaN(index) || index < 1) {
      console.error(chalk.red('Error:'), 'Invalid index. Use a positive number.');
      process.exit(1);
    }

    if (!newText || newText.trim().length === 0) {
      console.error(chalk.red('Error:'), 'Todo text cannot be empty');
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
    const oldText = todo.text;
    todo.text = newText.trim();

    const updatedContent = writeMarkdown(model);
    writeTodoFile(updatedContent);

    console.log(chalk.green('✓'), `Edited: "${oldText}" → "${newText.trim()}"`);
  } catch (error) {
    console.error(chalk.red('Error:'), (error as Error).message);
    process.exit(1);
  }
}
