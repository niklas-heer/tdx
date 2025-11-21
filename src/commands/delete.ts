import chalk from "chalk";
import { parseMarkdown } from "../todos/parser";
import { writeMarkdown } from "../todos/writer";
import {
  ensureTodoFileExists,
  readTodoFile,
  writeTodoFile,
} from "../fs/fileStore";

/**
 * Delete a todo by 1-based index.
 */
export function deleteTodo(indexStr: string): void {
  try {
    const index = Number.parseInt(indexStr, 10);

    if (Number.isNaN(index) || index < 1) {
      console.error(
        chalk.red("Error:"),
        "Todo index must be a positive number",
      );
      process.exit(1);
    }

    ensureTodoFileExists();
    const content = readTodoFile();
    const model = parseMarkdown(content);

    if (model.todos.length === 0 || index > model.todos.length) {
      console.error(
        chalk.red("Error:"),
        `Todo index ${index} out of range (1-${model.todos.length || 0})`,
      );
      process.exit(1);
    }

    const todo = model.todos[index - 1];
    const deletedLineNo = todo.lineNo;

    // Remove the line from lines array
    model.lines.splice(deletedLineNo, 1);

    // Remove the todo entry and re-index remaining todos
    model.todos.splice(index - 1, 1);
    for (let i = 0; i < model.todos.length; i++) {
      model.todos[i].index = i + 1;
      // Update lineNo for todos after the deleted line
      if (model.todos[i].lineNo > deletedLineNo) {
        model.todos[i].lineNo--;
      }
    }

    const updatedContent = writeMarkdown(model);
    writeTodoFile(updatedContent);

    console.log(chalk.green("✓"), `Deleted todo ${index}: ${todo.text}`);
  } catch (error) {
    console.error(chalk.red("Error:"), (error as Error).message);
    process.exit(1);
  }
}
