import chalk from "chalk";
import { parseMarkdown } from "../todos/parser";
import { readTodoFile, ensureTodoFileExists } from "../fs/fileStore";

export function list(): void {
  try {
    ensureTodoFileExists();
    const content = readTodoFile();
    const model = parseMarkdown(content);

    if (model.todos.length === 0) {
      console.log(
        chalk.gray('No todos yet. Use "tdx add <text>" to create one.'),
      );
      return;
    }

    for (const todo of model.todos) {
      const checkbox = todo.checked ? chalk.magenta("[✓]") : chalk.dim("[ ]");
      const text = todo.checked ? chalk.strikethrough(todo.text) : todo.text;
      console.log(`  ${todo.index}. ${checkbox} ${text}`);
    }
  } catch (error) {
    console.error(chalk.red("Error:"), (error as Error).message);
    process.exit(1);
  }
}
