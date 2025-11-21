#!/usr/bin/env bun

import { list } from "./commands/list";
import { add } from "./commands/add";
import { toggle } from "./commands/toggle";
import { edit } from "./commands/edit";
import { deleteTodo } from "./commands/delete";
import { launchTUI } from "./tui/index";
import {
  setTodoFilePath,
  todoFileExists,
  getTodoFilePath,
} from "./fs/fileStore";
import { checkForUpdates, showVersion, VERSION } from "./version";
import * as readline from "readline";

const args = process.argv.slice(2);

// Check for file path argument (first arg that's not a command)
let filePath: string | undefined;
let commandArgs = args;

// Check if first argument is a file path (not a known command)
const knownCommands = [
  "list",
  "add",
  "toggle",
  "edit",
  "delete",
  "help",
  "help-debug",
  "-h",
  "--help",
  "-v",
  "--version",
];
if (args.length > 0 && !knownCommands.includes(args[0])) {
  // First arg might be a file path
  if (
    args[0].endsWith(".md") ||
    args[0].includes("/") ||
    args[0].includes("\\")
  ) {
    filePath = args[0];
    commandArgs = args.slice(1);
  }
}

// Set custom file path if provided
if (filePath) {
  setTodoFilePath(filePath);
}

async function confirmCreate(path: string): Promise<boolean> {
  // If stdin is not a TTY (piped input), auto-confirm
  if (!process.stdin.isTTY) {
    return true;
  }

  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  return new Promise((resolve) => {
    rl.question(
      `File '${path}' does not exist. Create it? [y/N] `,
      (answer) => {
        rl.close();
        resolve(answer.toLowerCase() === "y" || answer.toLowerCase() === "yes");
      },
    );
  });
}

async function main() {
  // Check for updates (non-blocking, cached)
  await checkForUpdates();

  // If custom file path is set and file doesn't exist, ask for confirmation
  if (filePath && !todoFileExists()) {
    const confirmed = await confirmCreate(getTodoFilePath());
    if (!confirmed) {
      console.log("Aborted.");
      process.exit(0);
    }
  }

  if (commandArgs.length === 0) {
    // No arguments - launch TUI
    launchTUI();
  } else {
    const command = commandArgs[0];

    switch (command) {
      case "list":
        list();
        break;

      case "add":
        if (commandArgs.length < 2) {
          console.error('Usage: tdx [file] add "Text"');
          process.exit(1);
        }
        add(commandArgs.slice(1).join(" "));
        break;

      case "toggle":
        if (commandArgs.length < 2) {
          console.error("Usage: tdx [file] toggle <index>");
          process.exit(1);
        }
        toggle(commandArgs[1]);
        break;

      case "edit":
        if (commandArgs.length < 3) {
          console.error('Usage: tdx [file] edit <index> "New text"');
          process.exit(1);
        }
        edit(commandArgs[1], commandArgs.slice(2).join(" "));
        break;

      case "delete":
        if (commandArgs.length < 2) {
          console.error("Usage: tdx [file] delete <index>");
          process.exit(1);
        }
        deleteTodo(commandArgs[1]);
        break;

      case "help-debug":
        launchTUI(true);
        break;

      case "help":
      case "-h":
      case "--help":
        showHelp();
        break;

      case "-v":
      case "--version":
        showVersion();
        break;

      default:
        console.error(`Unknown command: ${command}`);
        console.error('Use "tdx help" for usage information');
        process.exit(1);
    }
  }
}

function showHelp() {
  console.log(`
tdx - A fast, lightweight todo manager

Usage:
  tdx [file]                    Launch interactive TUI
  tdx [file] list               List all todos
  tdx [file] add "Text"         Add a new todo
  tdx [file] toggle <index>     Toggle a todo (1-based index)
  tdx [file] edit <index> "Text" Edit a todo's text
  tdx [file] delete <index>     Delete a todo (1-based index)
  tdx help                      Show this help message
  tdx --version                 Show version information

File Path:
  If [file] is specified, use that file instead of todo.md in current directory.
  Will prompt for confirmation before creating a new file.

Interactive TUI Controls:
  j or Down              Move selection down
  k or Up                Move selection up
  Enter or Space         Toggle todo completion
  n                      Create a new todo
  e                      Edit selected todo text
  d                      Delete selected todo immediately
  u                      Undo last action
  [count]j               Move down by count (e.g., 5j moves down 5 lines)
  [count]k               Move up by count (e.g., 3k moves up 3 lines)
  q or Esc               Quit

Examples:
  tdx                          Use todo.md in current directory
  tdx project.md               Use project.md file
  tdx ~/notes/tasks.md         Use absolute path
  tdx test.md add "Buy milk"   Add to specific file
  tdx toggle 1
  tdx edit 2 "Buy cheese"
  tdx list
`);
}

main().catch((error) => {
  console.error("Error:", error.message);
  process.exit(1);
});
