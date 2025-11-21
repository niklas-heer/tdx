import React, { useState, useEffect, useRef } from "react";
import { Box, Text } from "ink";
import chalk from "chalk";
import { parseMarkdown } from "../todos/parser";
import { writeMarkdown } from "../todos/writer";
import {
  readTodoFile,
  writeTodoFile,
  ensureTodoFileExists,
} from "../fs/fileStore";
import { Todo } from "../todos/model";

// Render inline code blocks with styling
function renderInlineCode(text: string, isSelected: boolean): React.ReactNode {
  const parts: React.ReactNode[] = [];
  const regex = /`([^`]+)`/g;
  let lastIndex = 0;
  let match;
  let key = 0;

  while ((match = regex.exec(text)) !== null) {
    // Add text before the code block
    if (match.index > lastIndex) {
      const before = text.slice(lastIndex, match.index);
      parts.push(
        <Text key={key++}>
          {isSelected ? chalk.bold.whiteBright(before) : before}
        </Text>,
      );
    }

    // Add the code block with styling (cyan background, bold)
    const code = match[1];
    parts.push(
      <Text key={key++} backgroundColor="gray" color="white" bold>
        {` ${code} `}
      </Text>,
    );

    lastIndex = match.index + match[0].length;
  }

  // Add remaining text after the last code block
  if (lastIndex < text.length) {
    const after = text.slice(lastIndex);
    parts.push(
      <Text key={key++}>
        {isSelected ? chalk.bold.whiteBright(after) : after}
      </Text>,
    );
  }

  // If no code blocks found, return the original styled text
  if (parts.length === 0) {
    return isSelected ? chalk.bold.whiteBright(text) : text;
  }

  return parts;
}

interface AppState {
  todos: Todo[];
  lines: string[];
  selectedIndex: number;
  isLoading: boolean;
  error: string | null;
  inputMode: boolean;
  displayInputBuffer: string;
  editMode: boolean;
  showHelp: boolean;
  copyFeedback: boolean;
  moveMode: boolean;
  moveOriginalIndex: number;
}

interface AppProps {
  debug?: boolean;
}

export default function App({ debug = false }: AppProps) {
  const [state, setState] = useState<AppState>({
    todos: [],
    lines: [],
    selectedIndex: 0,
    isLoading: true,
    error: null,
    inputMode: false,
    displayInputBuffer: "",
    editMode: false,
    showHelp: debug,
    copyFeedback: false,
    moveMode: false,
    moveOriginalIndex: 0,
  });

  // Use refs to track mutable state
  const pendingChangeRef = useRef<boolean>(false);
  const inputBufferRef = useRef<string>("");
  const cursorPosRef = useRef<number>(0);
  const stateRef = useRef(state);
  const historyRef = useRef<{ todos: Todo[]; lines: string[] } | null>(null);
  const numberBufferRef = useRef<string>("");
  const keyBufferRef = useRef<string[]>([]);
  const handleKeyRef = useRef<(key: string) => void>(() => {});
  // Track if we're processing piped input
  const isPipedInputRef = useRef(!process.stdin.isTTY);
  const processingCompleteRef = useRef(false);
  const shouldExitRef = useRef(false);

  // Update handleKeyRef whenever state changes to ensure it has latest closures
  useEffect(() => {
    // This effect runs on every render to keep handleKeyRef up to date
    // The handleKey function is recreated on each render with fresh closures
  }, [state]);

  // Clear all buffers on mount to ensure clean state
  useEffect(() => {
    inputBufferRef.current = "";
    keyBufferRef.current = [];
    numberBufferRef.current = "";
  }, []);

  // Clear input buffer on mount to prevent any residual state
  useEffect(() => {
    inputBufferRef.current = "";
    keyBufferRef.current = [];
    numberBufferRef.current = "";
  }, []);

  // Keep stateRef in sync with state
  useEffect(() => {
    stateRef.current = state;
  }, [state]);

  // Handle exit outside of render cycle
  useEffect(() => {
    if (shouldExitRef.current) {
      process.exit(0);
    }
  });

  // Load todos on mount
  useEffect(() => {
    try {
      ensureTodoFileExists();
      const content = readTodoFile();
      const model = parseMarkdown(content);
      setState((prev) => ({
        todos: model.todos,
        lines: model.lines,
        selectedIndex: 0,
        isLoading: false,
        error: null,
        inputMode: false,
        displayInputBuffer: "",
        editMode: false,
        showHelp: prev.showHelp,
      }));
    } catch (error) {
      setState((prev) => ({
        ...prev,
        error: (error as Error).message,
        isLoading: false,
      }));
    }
  }, []);

  // Handle file persistence when todos change
  useEffect(() => {
    if (state.isLoading || state.error || !pendingChangeRef.current) {
      return;
    }

    pendingChangeRef.current = false;

    try {
      const updatedContent = writeMarkdown({
        lines: state.lines,
        todos: state.todos,
      });
      writeTodoFile(updatedContent);
    } catch (error) {
      console.error("Failed to save:", error);
    }
  }, [state.todos, state.lines]);

  // Define handleKey using useCallback to ensure it has latest state
  const handleKey = React.useCallback((key: string) => {
    // All key handling in a single setState to ensure we always have latest state
    setState((prev) => {
      // If still loading, buffer the key for later processing
      if (prev.isLoading || prev.error) {
        keyBufferRef.current.push(key);
        return prev;
      }

      // Handle Escape key - always available
      if (key === "\u001b") {
        if (prev.showHelp) {
          return { ...prev, showHelp: false };
        }
        if (prev.inputMode) {
          inputBufferRef.current = "";
          return {
            ...prev,
            inputMode: false,
            displayInputBuffer: "",
          };
        }
        if (prev.editMode) {
          inputBufferRef.current = "";
          return {
            ...prev,
            editMode: false,
            displayInputBuffer: "",
          };
        }
        if (prev.moveMode) {
          // Cancel move - restore original position
          const todos = [...prev.todos];
          const movedTodo = todos[prev.selectedIndex];
          todos.splice(prev.selectedIndex, 1);
          todos.splice(prev.moveOriginalIndex, 0, movedTodo);
          return {
            ...prev,
            todos,
            selectedIndex: prev.moveOriginalIndex,
            moveMode: false,
          };
        }
        shouldExitRef.current = true;
        return { ...prev }; // Return new object to trigger re-render
      }

      // Help menu always shows with ? key
      if (key === "?") {
        return {
          ...prev,
          showHelp: !prev.showHelp,
        };
      }

      // Don't process other keys if help is showing
      if (prev.showHelp) {
        return prev;
      }

      // Move mode handling
      if (prev.moveMode) {
        if (key === "\r" || key === "\n") {
          // Confirm move - update line numbers and save
          const updatedTodos = prev.todos.map((todo, idx) => ({
            ...todo,
            lineNo: idx,
            index: idx + 1,
          }));
          const updatedLines = updatedTodos.map(
            (todo) => `- ${todo.checked ? "[x]" : "[ ]"} ${todo.text}`,
          );
          pendingChangeRef.current = true;
          return {
            ...prev,
            todos: updatedTodos,
            lines: updatedLines,
            moveMode: false,
          };
        }
        if (key === "j" || key === "J" || key === "\u001b[B") {
          // Move down
          if (prev.selectedIndex < prev.todos.length - 1) {
            const todos = [...prev.todos];
            const temp = todos[prev.selectedIndex];
            todos[prev.selectedIndex] = todos[prev.selectedIndex + 1];
            todos[prev.selectedIndex + 1] = temp;
            return {
              ...prev,
              todos,
              selectedIndex: prev.selectedIndex + 1,
            };
          }
          return prev;
        }
        if (key === "k" || key === "K" || key === "\u001b[A") {
          // Move up
          if (prev.selectedIndex > 0) {
            const todos = [...prev.todos];
            const temp = todos[prev.selectedIndex];
            todos[prev.selectedIndex] = todos[prev.selectedIndex - 1];
            todos[prev.selectedIndex - 1] = temp;
            return {
              ...prev,
              todos,
              selectedIndex: prev.selectedIndex - 1,
            };
          }
          return prev;
        }
        return prev;
      }

      // Process the rest of the key handling
      // Use prev for all state checks to ensure we have latest state
      const currentState = prev;

      // In input mode or edit mode, only handle Enter and Backspace
      if (currentState.inputMode || currentState.editMode) {
        if (key === "\r" || key === "\n") {
          const trimmedText = inputBufferRef.current.trim();
          // Clear buffer immediately after processing
          const textToSave = trimmedText;
          inputBufferRef.current = "";

          if (textToSave.length > 0) {
            if (currentState.editMode) {
              // Update existing todo
              // Use the prev parameter which has latest state
              const updatedTodos = [...prev.todos];
              const selectedTodo = updatedTodos[prev.selectedIndex];

              // Safety check - if selectedTodo is undefined, exit edit mode
              if (!selectedTodo) {
                return {
                  ...prev,
                  editMode: false,
                  displayInputBuffer: "",
                };
              }
              updatedTodos[prev.selectedIndex] = {
                ...selectedTodo,
                text: textToSave,
              };

              const updatedLines = [...prev.lines];
              updatedLines[selectedTodo.lineNo] =
                `- ${selectedTodo.checked ? "[x]" : "[ ]"} ${textToSave}`;

              pendingChangeRef.current = true;

              return {
                ...prev,
                todos: updatedTodos,
                lines: updatedLines,
                editMode: false,
                displayInputBuffer: "",
              };
            } else {
              // Create new todo
              const newTodo: Todo = {
                index: prev.todos.length + 1,
                checked: false,
                text: textToSave,
                lineNo: prev.lines.length,
              };

              const updatedTodos = [...prev.todos, newTodo];
              const updatedLines = [...prev.lines, `- [ ] ${textToSave}`];

              pendingChangeRef.current = true;

              return {
                ...prev,
                todos: updatedTodos,
                lines: updatedLines,
                selectedIndex: updatedTodos.length - 1,
                inputMode: false,
                displayInputBuffer: "",
              };
            }
          } else {
            // Empty input - for piped input, stay in input mode to wait for text
            // For interactive use, exit input mode
            if (isPipedInputRef.current) {
              // In piped mode, empty Enter likely means we're still waiting for text
              // Just ignore this Enter and stay in input mode
              return { ...prev }; // Return new object to ensure re-render
            }
            // Interactive mode - exit on empty input
            inputBufferRef.current = "";
            return {
              ...prev,
              inputMode: false,
              editMode: false,
              displayInputBuffer: "",
            };
          }
        } else if (key === "\u007f" || key === "\b") {
          // Backspace - delete character before cursor
          if (cursorPosRef.current > 0) {
            const before = inputBufferRef.current.slice(
              0,
              cursorPosRef.current - 1,
            );
            const after = inputBufferRef.current.slice(cursorPosRef.current);
            inputBufferRef.current = before + after;
            cursorPosRef.current--;
          }
          return {
            ...prev,
            displayInputBuffer: inputBufferRef.current,
          };
        } else if (key === "\u001b[3~") {
          // Delete key - delete character at cursor
          if (cursorPosRef.current < inputBufferRef.current.length) {
            const before = inputBufferRef.current.slice(
              0,
              cursorPosRef.current,
            );
            const after = inputBufferRef.current.slice(
              cursorPosRef.current + 1,
            );
            inputBufferRef.current = before + after;
          }
          return {
            ...prev,
            displayInputBuffer: inputBufferRef.current,
          };
        } else if (key === "\u001b[D") {
          // Left arrow - move cursor left
          if (cursorPosRef.current > 0) {
            cursorPosRef.current--;
          }
          return { ...prev, displayInputBuffer: inputBufferRef.current };
        } else if (key === "\u001b[C") {
          // Right arrow - move cursor right
          if (cursorPosRef.current < inputBufferRef.current.length) {
            cursorPosRef.current++;
          }
          return { ...prev, displayInputBuffer: inputBufferRef.current };
        } else if (
          key === "\u001b[H" ||
          key === "\u001b[1~" ||
          key === "\u001bOH" ||
          key === "\u0001"
        ) {
          // Home or Ctrl+A - move to start
          cursorPosRef.current = 0;
          return { ...prev, displayInputBuffer: inputBufferRef.current };
        } else if (
          key === "\u001b[F" ||
          key === "\u001b[4~" ||
          key === "\u001bOF" ||
          key === "\u0005"
        ) {
          // End or Ctrl+E - move to end
          cursorPosRef.current = inputBufferRef.current.length;
          return { ...prev, displayInputBuffer: inputBufferRef.current };
        } else if (key === "\u001bb" || key === "\u001b[1;3D") {
          // Option+Left (Alt+Left) - move word left
          let pos = cursorPosRef.current;
          // Skip spaces
          while (pos > 0 && inputBufferRef.current[pos - 1] === " ") pos--;
          // Skip word
          while (pos > 0 && inputBufferRef.current[pos - 1] !== " ") pos--;
          cursorPosRef.current = pos;
          return { ...prev, displayInputBuffer: inputBufferRef.current };
        } else if (key === "\u001bf" || key === "\u001b[1;3C") {
          // Option+Right (Alt+Right) - move word right
          let pos = cursorPosRef.current;
          const len = inputBufferRef.current.length;
          // Skip word
          while (pos < len && inputBufferRef.current[pos] !== " ") pos++;
          // Skip spaces
          while (pos < len && inputBufferRef.current[pos] === " ") pos++;
          cursorPosRef.current = pos;
          return { ...prev, displayInputBuffer: inputBufferRef.current };
        } else if (key === "\u001b[1;2D") {
          // Cmd+Left (Shift+Left in some terminals) - move to start of line
          cursorPosRef.current = 0;
          return { ...prev, displayInputBuffer: inputBufferRef.current };
        } else if (key === "\u001b[1;2C") {
          // Cmd+Right (Shift+Right in some terminals) - move to end of line
          cursorPosRef.current = inputBufferRef.current.length;
          return { ...prev, displayInputBuffer: inputBufferRef.current };
        } else if (key.length === 1 && key.charCodeAt(0) >= 32) {
          // Regular printable character - insert at cursor
          const before = inputBufferRef.current.slice(0, cursorPosRef.current);
          const after = inputBufferRef.current.slice(cursorPosRef.current);
          inputBufferRef.current = before + key + after;
          cursorPosRef.current++;
          return {
            ...prev,
            displayInputBuffer: inputBufferRef.current,
          };
        }
        // Ignore all other keys in input/edit mode
        return prev;
      }

      if (key === "\u0003") {
        // Ctrl+C
        shouldExitRef.current = true;
        return { ...prev }; // Return new object to trigger re-render
      }

      if (key === "u" || key === "U") {
        // Undo last action
        // Use setState callback to get latest state
        if (historyRef.current) {
          const previousState = historyRef.current;
          historyRef.current = null;
          pendingChangeRef.current = true;

          return {
            ...prev,
            todos: previousState.todos,
            lines: previousState.lines,
            selectedIndex: Math.min(
              prev.selectedIndex,
              previousState.todos.length - 1,
            ),
          };
        }
        return prev;
      }

      if (key === "e" || key === "E") {
        // Enter edit mode
        // Use setState callback to get latest state
        if (prev.todos.length === 0) {
          return prev;
        }
        inputBufferRef.current = prev.todos[prev.selectedIndex].text;
        cursorPosRef.current = inputBufferRef.current.length; // Cursor at end
        // Save state for undo before entering edit mode (deep copy todos array)
        historyRef.current = {
          todos: prev.todos.map((todo) => ({ ...todo })),
          lines: [...prev.lines],
        };
        return {
          ...prev,
          editMode: true,
          displayInputBuffer: inputBufferRef.current,
        };
      }

      if (key === "c" || key === "C") {
        // Copy todo text to clipboard
        if (prev.todos.length === 0) {
          return prev;
        }
        const textToCopy = prev.todos[prev.selectedIndex].text;
        // Use pbcopy on macOS with printf to avoid newline
        import("child_process").then(({ exec }) => {
          exec(`printf %s ${JSON.stringify(textToCopy)} | pbcopy`);
        });
        // Show feedback and clear after timeout
        setTimeout(() => {
          setState((p) => ({ ...p, copyFeedback: false }));
        }, 1500);
        return { ...prev, copyFeedback: true };
      }

      if (key === "m" || key === "M") {
        // Enter move mode
        if (prev.todos.length === 0) {
          return prev;
        }
        // Save state for undo before entering move mode
        historyRef.current = {
          todos: prev.todos.map((todo) => ({ ...todo })),
          lines: [...prev.lines],
        };
        return {
          ...prev,
          moveMode: true,
          moveOriginalIndex: prev.selectedIndex,
        };
      }

      // Handle number input for vim-style relative jumping (e.g., 5j, 3k)
      if (key >= "1" && key <= "9") {
        numberBufferRef.current += key;
        return prev;
      }

      // If we have a number buffer, check for j/k to apply relative movement
      if (numberBufferRef.current) {
        const count = parseInt(numberBufferRef.current, 10);
        numberBufferRef.current = "";

        if (key === "j" || key === "J") {
          // Move down by count
          const newIndex = Math.min(
            prev.selectedIndex + count,
            prev.todos.length - 1,
          );
          if (newIndex !== prev.selectedIndex) {
            return { ...prev, selectedIndex: newIndex };
          }
          return prev;
        } else if (key === "k" || key === "K") {
          // Move up by count
          const newIndex = Math.max(prev.selectedIndex - count, 0);
          if (newIndex !== prev.selectedIndex) {
            return { ...prev, selectedIndex: newIndex };
          }
          return prev;
        }
        // If it's not j or k, clear the buffer and continue processing this key
      }

      if (key === "n" || key === "N") {
        // Enter input mode
        // Use setState callback to get latest state
        inputBufferRef.current = "";
        cursorPosRef.current = 0; // Cursor at start for new input
        numberBufferRef.current = "";
        // Save state for undo before entering input mode (deep copy todos array)
        historyRef.current = {
          todos: prev.todos.map((todo) => ({ ...todo })),
          lines: [...prev.lines],
        };
        return {
          ...prev,
          inputMode: true,
          displayInputBuffer: "",
        };
      }

      if (key === "d") {
        // Clear number buffer when executing other commands
        numberBufferRef.current = "";

        // Use setState callback to get latest state
        if (prev.todos.length === 0) {
          return prev;
        }

        // Save state for undo (deep copy todos array)
        historyRef.current = {
          todos: prev.todos.map((todo) => ({ ...todo })),
          lines: [...prev.lines],
        };

        const deletedTodo = prev.todos[prev.selectedIndex];
        const remainingTodos = prev.todos
          .filter((_, idx) => idx !== prev.selectedIndex)
          .map((todo, idx) => ({
            ...todo,
            index: idx + 1,
            lineNo:
              todo.lineNo > deletedTodo.lineNo ? todo.lineNo - 1 : todo.lineNo,
          }));
        const remainingLines = prev.lines.filter(
          (_, lineIndex) => lineIndex !== deletedTodo.lineNo,
        );
        const nextSelectedIndex =
          remainingTodos.length === 0
            ? 0
            : Math.min(prev.selectedIndex, remainingTodos.length - 1);

        pendingChangeRef.current = true;

        return {
          ...prev,
          todos: remainingTodos,
          lines: remainingLines,
          selectedIndex: nextSelectedIndex,
        };
      }

      // Clear number buffer when executing other commands
      numberBufferRef.current = "";

      // Navigation and toggle handling
      let newIndex = prev.selectedIndex;
      let indexChanged = false;

      if (key === "j" || key === "\x1b[B") {
        if (prev.todos.length === 0) {
          return prev;
        }
        newIndex = Math.min(prev.selectedIndex + 1, prev.todos.length - 1);
        indexChanged = true;
      } else if (key === "k" || key === "\x1b[A") {
        if (prev.todos.length === 0) {
          return prev;
        }
        newIndex = Math.max(prev.selectedIndex - 1, 0);
        indexChanged = true;
      } else if (key === "\r" || key === "\n" || key === " ") {
        if (prev.todos.length === 0) {
          return prev;
        }

        // Save state for undo (deep copy todos array)
        historyRef.current = {
          todos: prev.todos.map((todo) => ({ ...todo })),
          lines: [...prev.lines],
        };

        const updatedTodos = [...prev.todos];
        updatedTodos[prev.selectedIndex].checked =
          !updatedTodos[prev.selectedIndex].checked;

        pendingChangeRef.current = true;

        return {
          ...prev,
          todos: updatedTodos,
        };
      }

      if (indexChanged && newIndex !== prev.selectedIndex) {
        return { ...prev, selectedIndex: newIndex };
      }
      return prev;
    });
  }, []); // Empty dependency array - handleKey uses stateRef for latest state

  // Setup key listeners - run immediately, don't wait for loading
  useEffect(() => {
    // Store handleKey in ref for later access
    // This ensures handleKeyRef always points to the latest version
    handleKeyRef.current = handleKey;

    // Enable raw mode for TTY, but always listen for input (supports piped input)
    if (process.stdin.isTTY) {
      process.stdin.setRawMode(true);
    }
    process.stdin.resume();
    process.stdin.setEncoding("utf-8");

    const dataListener = (chunk: Buffer) => {
      // Process input, handling escape sequences as single keys
      const input = chunk.toString();
      const keys: string[] = [];
      let i = 0;
      while (i < input.length) {
        // Check for escape sequences (arrow keys, etc.)
        if (
          input[i] === "\x1b" &&
          i + 2 < input.length &&
          input[i + 1] === "["
        ) {
          // This is an escape sequence like \x1b[A (up arrow)
          keys.push(input.slice(i, i + 3));
          i += 3;
        } else {
          keys.push(input[i]);
          i++;
        }
      }
      // Process all keys in next tick to avoid render cycle issues
      process.nextTick(() => {
        keys.forEach((key) => handleKey(key));
      });
    };

    process.stdin.on("data", dataListener);

    // Handle piped input: exit after processing if not a TTY
    if (!process.stdin.isTTY) {
      process.stdin.on("end", () => {
        // Wait for processing to complete before exiting
        const checkAndExit = () => {
          if (processingCompleteRef.current) {
            // Give time for final file operations to complete
            setTimeout(() => {
              process.exit(0);
            }, 300);
          } else {
            // Still processing, check again
            setTimeout(checkAndExit, 100);
          }
        };
        // Delay initial check to allow buffered keys to start processing
        setTimeout(checkAndExit, 500);
      });
    }

    // Store handleKey in ref for later access
    // This ensures handleKeyRef always points to the latest version
    handleKeyRef.current = handleKey;

    return () => {
      if (process.stdin.isTTY) {
        process.stdin.setRawMode(false);
      }
      process.stdin.pause();
      process.stdin.removeListener("data", dataListener);
    };
  }, [handleKey]); // Re-run when handleKey changes to update the ref

  // Process buffered keys after loading completes
  useEffect(() => {
    if (!state.isLoading && !state.error && keyBufferRef.current.length > 0) {
      // Clear all buffers before processing buffered keys
      inputBufferRef.current = "";
      numberBufferRef.current = "";

      // Process all buffered keys
      const bufferedKeys = [...keyBufferRef.current];
      keyBufferRef.current = [];

      // Process keys with delays to allow React state updates to propagate
      setTimeout(() => {
        let i = 0;
        const processNextKey = () => {
          if (i < bufferedKeys.length) {
            try {
              handleKeyRef.current(bufferedKeys[i]);
            } catch (err) {
              // Ignore key processing errors
            }
            i++;
            // Longer delay between keys to allow state updates to propagate
            // Mode changes (n, e) need more time than regular characters
            const delay = 20;
            setTimeout(processNextKey, delay);
          } else {
            // Mark processing as complete
            processingCompleteRef.current = true;
            // If piped input, exit after processing
            if (isPipedInputRef.current) {
              setTimeout(() => {
                process.exit(0);
              }, 300);
            }
          }
        };
        processNextKey();
      }, 100);
    }
  }, [state.isLoading, state.error]);

  const {
    todos,
    selectedIndex,
    isLoading,
    error,
    inputMode,
    editMode,
    displayInputBuffer,
    showHelp,
    copyFeedback,
    moveMode,
  } = state;

  if (isLoading) {
    return <Text>Loading todos...</Text>;
  }

  if (error) {
    return <Text color="red">Error: {error}</Text>;
  }

  if (showHelp) {
    // Define help data in a structured format
    const helpSections = [
      {
        title: "NAVIGATION",
        items: [
          { shortcut: "j / ↓", description: "Move down" },
          { shortcut: "k / ↑", description: "Move up" },
          { shortcut: "5j / 3k", description: "Jump by count" },
        ],
      },
      {
        title: "EDITING",
        items: [
          { shortcut: "␣", description: "Toggle todo" },
          { shortcut: "n", description: "New todo" },
          { shortcut: "e", description: "Edit text" },
          { shortcut: "d", description: "Delete" },
          { shortcut: "c", description: "Copy text" },
          { shortcut: "m", description: "Move" },
        ],
      },
      {
        title: "OTHER",
        items: [
          { shortcut: "u", description: "Undo" },
          { shortcut: "?", description: "Toggle help" },
          { shortcut: "esc", description: "Quit" },
        ],
      },
    ];

    // Calculate column widths per section
    const sectionWidths = helpSections.map((section) => {
      const shortcutWidth =
        Math.max(...section.items.map((item) => item.shortcut.length)) + 1;
      const descriptionWidth = Math.max(
        ...section.items.map((item) => item.description.length),
      );
      return {
        shortcut: shortcutWidth,
        description: descriptionWidth,
        total: shortcutWidth + descriptionWidth,
      };
    });

    const totalWidth =
      sectionWidths.reduce((sum, s) => sum + s.total, 0) +
      (helpSections.length - 1) * 2;

    // Helper functions
    const pad = (text: string, width: number): string => {
      return text + " ".repeat(Math.max(0, width - text.length));
    };

    const center = (text: string, width: number): string => {
      const padding = Math.max(0, width - text.length);
      const left = Math.floor(padding / 2);
      const right = padding - left;
      return " ".repeat(left) + text + " ".repeat(right);
    };

    const border = "─".repeat(totalWidth);

    return (
      <Box flexDirection="column">
        {/* Title row */}
        <Box>
          {helpSections.map((section, idx) => (
            <Box
              key={section.title}
              width={sectionWidths[idx].total}
              marginRight={idx < helpSections.length - 1 ? 2 : 0}
            >
              <Text bold color="cyan">
                {center(section.title, sectionWidths[idx].total)}
              </Text>
            </Box>
          ))}
        </Box>

        {/* Border */}
        <Text color="gray">{border}</Text>

        {/* Content rows */}
        {Array.from({
          length: Math.max(...helpSections.map((s) => s.items.length)),
        }).map((_, rowIdx) => (
          <Box key={`row-${rowIdx}`}>
            {helpSections.map((section, colIdx) => {
              const item = section.items[rowIdx];
              return (
                <Box
                  key={`${section.title}-${rowIdx}`}
                  width={sectionWidths[colIdx].total}
                  marginRight={colIdx < helpSections.length - 1 ? 2 : 0}
                >
                  {item ? (
                    <Text>
                      <Text color="cyan">
                        {pad(item.shortcut, sectionWidths[colIdx].shortcut)}
                      </Text>
                      <Text>{item.description}</Text>
                    </Text>
                  ) : (
                    <Text></Text>
                  )}
                </Box>
              );
            })}
          </Box>
        ))}

        {/* Border */}
        <Text color="gray">{border}</Text>

        {/* Footer */}
        <Text color="gray">Press ? or esc to close help</Text>
      </Box>
    );
  }

  if (inputMode) {
    const beforeCursor = displayInputBuffer.slice(0, cursorPosRef.current);
    const afterCursor = displayInputBuffer.slice(cursorPosRef.current);
    return (
      <Box flexDirection="column">
        <Box>
          <Text color="cyan">➜ </Text>
          <Text color="green">[ ] </Text>
          <Text>{beforeCursor}</Text>
          <Text backgroundColor="white" color="black">
            {afterCursor[0] || " "}
          </Text>
          <Text>{afterCursor.slice(1)}</Text>
        </Box>
        <Text color="gray">(Press Enter to confirm, Esc to cancel)</Text>
      </Box>
    );
  }

  if (editMode) {
    const beforeCursor = displayInputBuffer.slice(0, cursorPosRef.current);
    const afterCursor = displayInputBuffer.slice(cursorPosRef.current);
    return (
      <Box flexDirection="column">
        <Box>
          <Text color="cyan">➜ </Text>
          <Text color="green">[ ] </Text>
          <Text>{beforeCursor}</Text>
          <Text backgroundColor="white" color="black">
            {afterCursor[0] || " "}
          </Text>
          <Text>{afterCursor.slice(1)}</Text>
        </Box>
        <Text color="gray">(Press Enter to confirm, Esc to cancel)</Text>
      </Box>
    );
  }

  if (todos.length === 0) {
    return (
      <Box flexDirection="column">
        <Text color="gray">
          No todos yet. Press "n" to create one or use "tdx add &lt;text&gt;".
        </Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      {todos.map((todo, index) => {
        const isSelected = index === selectedIndex;
        const checkbox = todo.checked ? chalk.magenta("[✓]") : chalk.dim("[ ]");
        const arrow = isSelected
          ? moveMode
            ? chalk.yellow("≡")
            : chalk.cyan("➜")
          : " ";
        const renderedText = renderInlineCode(todo.text, isSelected);
        const relativeIndex = index - selectedIndex;
        let relativeDisplay = "";
        if (relativeIndex === 0) {
          relativeDisplay = "";
        } else if (relativeIndex > 0) {
          relativeDisplay = `+${relativeIndex}`;
        } else {
          relativeDisplay = `${relativeIndex}`;
        }

        // Calculate prefix width for alignment: "±XX ➜ [x] " = 3 + 1 + 1 + 3 + 1 = 9 chars
        const prefixWidth = 9;

        return (
          <Box key={todo.lineNo} flexDirection="row">
            <Box width={prefixWidth} flexShrink={0}>
              <Text color="gray" dimColor>
                {relativeDisplay.padEnd(3)}
              </Text>
              <Text>{arrow}</Text>
              <Text> </Text>
              <Text>{checkbox}</Text>
              <Text> </Text>
            </Box>
            <Box flexGrow={1}>
              <Text wrap="wrap">{renderedText}</Text>
            </Box>
          </Box>
        );
      })}
      <Box marginTop={1}>
        {copyFeedback ? (
          <Text color="green">Copied to clipboard!</Text>
        ) : moveMode ? (
          <Text color="yellow">
            Moving: <Text color="cyan">j</Text>/<Text color="cyan">k</Text>
            {" move  |  "}
            <Text color="cyan">enter</Text>
            {" confirm  |  "}
            <Text color="cyan">esc</Text>
            {" cancel"}
          </Text>
        ) : (
          <Text color="gray">
            <Text color="cyan">?</Text>
            {" help  |  "}
            <Text color="cyan">j</Text>/<Text color="cyan">k</Text>
            {" nav  |  "}
            <Text color="cyan">n</Text>
            {" new  |  "}
            <Text color="cyan">␣</Text>
            {" toggle  |  "}
            <Text color="cyan">esc</Text>
            {" quit"}
          </Text>
        )}
      </Box>
    </Box>
  );
}
