#!/usr/bin/env bun

import { list } from "./commands/list";
import { add } from "./commands/add";
import { toggle } from "./commands/toggle";
import { edit } from "./commands/edit";
import { deleteTodo } from "./commands/delete";
import { launchTUI } from "./tui/index";

const args = process.argv.slice(2);

if (args.length === 0) {
  // No arguments - launch TUI
  launchTUI();
} else {
  const command = args[0];

  switch (command) {
    case "list":
      list();
      break;

    case "add":
      if (args.length < 2) {
        console.error('Usage: tdx add "Text"');
        process.exit(1);
      }
      add(args.slice(1).join(" "));
      break;

    case "toggle":
      if (args.length < 2) {
        console.error("Usage: tdx toggle <index>");
        process.exit(1);
      }
      toggle(args[1]);
      break;

    case "edit":
      if (args.length < 3) {
        console.error('Usage: tdx edit <index> "New text"');
        process.exit(1);
      }
      edit(args[1], args.slice(2).join(" "));
      break;

    case "delete":
      if (args.length < 2) {
        console.error("Usage: tdx delete <index>");
        process.exit(1);
      }
      deleteTodo(args[1]);
      break;

    case "help":
    case "-h":
    case "--help":
      showHelp();
      break;

    default:
      console.error(`Unknown command: ${command}`);
      console.error('Use "tdx help" for usage information');
      process.exit(1);
  }
}

function showHelp() {
  console.log(`
tdx - A fast, lightweight todo manager

Usage:
  tdx                    Launch interactive TUI
  tdx list               List all todos
  tdx add "Text"         Add a new todo
  tdx toggle <index>     Toggle a todo (1-based index)
  tdx edit <index> "Text" Edit a todo's text
  tdx delete <index>     Delete a todo (1-based index)
  tdx help               Show this help message

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
  tdx add "Buy milk"
  tdx toggle 1
  tdx edit 2 "Buy cheese"
  tdx list
`);
}
