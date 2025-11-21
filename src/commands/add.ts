import chalk from "chalk";
import { parseMarkdown } from "../todos/parser";
import { writeMarkdown } from "../todos/writer";
import {
  readTodoFile,
  writeTodoFile,
  ensureTodoFileExists,
} from "../fs/fileStore";

export function add(text: string): void {
  try {
    if (!text || text.trim().length === 0) {
      console.error(chalk.red("Error:"), "Todo text cannot be empty");
      process.exit(1);
    }

    ensureTodoFileExists();
    const content = readTodoFile();
    const model = parseMarkdown(content);

    // Add new line with the new todo
    model.lines.push(`- [ ] ${text.trim()}`);

    const updatedContent = writeMarkdown(model);
    writeTodoFile(updatedContent);

    console.log(chalk.green("✓"), `Added: ${text.trim()}`);
  } catch (error) {
    console.error(chalk.red("Error:"), (error as Error).message);
    process.exit(1);
  }
}
