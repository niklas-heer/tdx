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
}

export default function App() {
  const [state, setState] = useState<AppState>({
    todos: [],
    lines: [],
    selectedIndex: 0,
    isLoading: true,
    error: null,
    inputMode: false,
    displayInputBuffer: "",
    editMode: false,
    showHelp: false,
  });

  // Use refs to track mutable state
  const pendingChangeRef = useRef<boolean>(false);
  const inputBufferRef = useRef<string>("");
  const stateRef = useRef(state);
  const historyRef = useRef<{ todos: Todo[]; lines: string[] } | null>(null);
  const numberBufferRef = useRef<string>("");

  // Keep stateRef in sync with state
  useEffect(() => {
    stateRef.current = state;
  }, [state]);

  // Load todos on mount
  useEffect(() => {
    try {
      ensureTodoFileExists();
      const content = readTodoFile();
      const model = parseMarkdown(content);
      setState({
        todos: model.todos,
        lines: model.lines,
        selectedIndex: 0,
        isLoading: false,
        error: null,
        inputMode: false,
        displayInputBuffer: "",
        editMode: false,
        showHelp: false,
      });
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

  // Setup key listeners
  useEffect(() => {
    if (state.isLoading || state.error) return;

    const handleKey = (key: string) => {
      const currentState = stateRef.current;

      // Handle Escape key - always available
      if (key === "\u001b") {
        if (currentState.inputMode) {
          inputBufferRef.current = "";
          setState((prev) => ({
            ...prev,
            inputMode: false,
            displayInputBuffer: "",
          }));
          return;
        }
        if (currentState.editMode) {
          inputBufferRef.current = "";
          setState((prev) => ({
            ...prev,
            editMode: false,
            displayInputBuffer: "",
          }));
          return;
        }
        process.exit(0);
        return;
      }

      // Help menu always shows with ? key
      if (key === "?") {
        setState((prev) => ({
          ...prev,
          showHelp: !prev.showHelp,
        }));
        return;
      }

      // Help menu closes with q or Escape
      if (
        currentState.showHelp &&
        (key === "q" || key === "Q" || key === "\u001b")
      ) {
        setState((prev) => ({
          ...prev,
          showHelp: false,
        }));
        return;
      }

      // Don't process other keys if help is showing
      if (currentState.showHelp) {
        return;
      }

      // In input mode or edit mode, only handle Enter and Backspace
      if (currentState.inputMode || currentState.editMode) {
        if (key === "\r" || key === "\n") {
          const trimmedText = inputBufferRef.current.trim();
          if (trimmedText.length > 0) {
            if (currentState.editMode) {
              // Update existing todo
              setState((prev) => {
                const updatedTodos = [...prev.todos];
                const selectedTodo = updatedTodos[prev.selectedIndex];
                updatedTodos[prev.selectedIndex] = {
                  ...selectedTodo,
                  text: trimmedText,
                };

                const updatedLines = [...prev.lines];
                updatedLines[selectedTodo.lineNo] =
                  `- ${selectedTodo.checked ? "[x]" : "[ ]"} ${trimmedText}`;

                pendingChangeRef.current = true;
                inputBufferRef.current = "";

                return {
                  ...prev,
                  todos: updatedTodos,
                  lines: updatedLines,
                  editMode: false,
                  displayInputBuffer: "",
                };
              });
            } else {
              // Create new todo
              setState((prev) => {
                const newTodo: Todo = {
                  index: prev.todos.length + 1,
                  checked: false,
                  text: trimmedText,
                  lineNo: prev.lines.length,
                };

                const updatedTodos = [...prev.todos, newTodo];
                const updatedLines = [...prev.lines, `- [ ] ${trimmedText}`];

                pendingChangeRef.current = true;
                inputBufferRef.current = "";

                return {
                  ...prev,
                  todos: updatedTodos,
                  lines: updatedLines,
                  selectedIndex: updatedTodos.length - 1,
                  inputMode: false,
                  displayInputBuffer: "",
                };
              });
            }
          } else {
            // Empty input, just exit
            setState((prev) => ({
              ...prev,
              inputMode: false,
              editMode: false,
              displayInputBuffer: "",
            }));
            inputBufferRef.current = "";
          }
          return;
        } else if (key === "\u007f" || key === "\b") {
          // Backspace
          inputBufferRef.current = inputBufferRef.current.slice(0, -1);
          setState((prev) => ({
            ...prev,
            displayInputBuffer: inputBufferRef.current,
          }));
          return;
        } else if (key.length === 1 && key.charCodeAt(0) >= 32) {
          // Regular printable character
          inputBufferRef.current += key;
          setState((prev) => ({
            ...prev,
            displayInputBuffer: inputBufferRef.current,
          }));
          return;
        } else if (key.length === 1 && key.charCodeAt(0) >= 32) {
          // Regular printable character
          inputBufferRef.current += key;
          setState((prev) => ({
            ...prev,
            displayInputBuffer: inputBufferRef.current,
          }));
          return;
        }
        // Ignore all other keys in input/edit mode
        return;
      }

      // Normal mode key handling (when not in input mode)
      if (key === "q" || key === "Q") {
        process.exit(0);
        return;
      }

      if (key === "\u0003") {
        // Ctrl+C
        process.exit(0);
        return;
      }

      if (key === "u" || key === "U") {
        // Undo last action
        if (historyRef.current) {
          const previousState = historyRef.current;
          historyRef.current = null;
          pendingChangeRef.current = true;

          setState((prev) => ({
            ...prev,
            todos: previousState.todos,
            lines: previousState.lines,
            selectedIndex: Math.min(
              prev.selectedIndex,
              previousState.todos.length - 1,
            ),
          }));
        }
        return;
      }

      if (key === "e" || key === "E") {
        // Enter edit mode
        if (currentState.todos.length === 0) {
          return;
        }
        inputBufferRef.current =
          currentState.todos[currentState.selectedIndex].text;
        // Save state for undo before entering edit mode (deep copy todos array)
        historyRef.current = {
          todos: currentState.todos.map((todo) => ({ ...todo })),
          lines: [...currentState.lines],
        };
        setState((prev) => ({
          ...prev,
          editMode: true,
          displayInputBuffer: inputBufferRef.current,
        }));
        return;
      }

      // Handle number input for vim-style relative jumping (e.g., 5j, 3k)
      if (key >= "1" && key <= "9") {
        numberBufferRef.current += key;
        return;
      }

      // If we have a number buffer, check for j/k to apply relative movement
      if (numberBufferRef.current) {
        const count = parseInt(numberBufferRef.current, 10);
        numberBufferRef.current = "";

        if (key === "j" || key === "J") {
          // Move down by count
          const newIndex = Math.min(
            currentState.selectedIndex + count,
            currentState.todos.length - 1,
          );
          if (newIndex !== currentState.selectedIndex) {
            setState((prev) => ({
              ...prev,
              selectedIndex: newIndex,
            }));
          }
          return;
        } else if (key === "k" || key === "K") {
          // Move up by count
          const newIndex = Math.max(currentState.selectedIndex - count, 0);
          if (newIndex !== currentState.selectedIndex) {
            setState((prev) => ({
              ...prev,
              selectedIndex: newIndex,
            }));
          }
          return;
        }
        // If it's not j or k, clear the buffer and continue processing this key
      }

      if (key === "n" || key === "N") {
        // Enter input mode
        inputBufferRef.current = "";
        // Save state for undo before entering input mode (deep copy todos array)
        historyRef.current = {
          todos: currentState.todos.map((todo) => ({ ...todo })),
          lines: [...currentState.lines],
        };
        setState((prev) => ({
          ...prev,
          inputMode: true,
          displayInputBuffer: "",
        }));
        return;
      }

      if (key === "d") {
        // Clear number buffer when executing other commands
        numberBufferRef.current = "";

        if (currentState.todos.length === 0) {
          return;
        }

        // Save state for undo (deep copy todos array)
        historyRef.current = {
          todos: currentState.todos.map((todo) => ({ ...todo })),
          lines: [...currentState.lines],
        };

        const deletedTodo = currentState.todos[currentState.selectedIndex];
        const remainingTodos = currentState.todos
          .filter((_, idx) => idx !== currentState.selectedIndex)
          .map((todo, idx) => ({
            ...todo,
            index: idx + 1,
            lineNo:
              todo.lineNo > deletedTodo.lineNo ? todo.lineNo - 1 : todo.lineNo,
          }));
        const remainingLines = currentState.lines.filter(
          (_, lineIndex) => lineIndex !== deletedTodo.lineNo,
        );
        const nextSelectedIndex =
          remainingTodos.length === 0
            ? 0
            : Math.min(currentState.selectedIndex, remainingTodos.length - 1);

        pendingChangeRef.current = true;

        setState((prev) => ({
          ...prev,
          todos: remainingTodos,
          lines: remainingLines,
          selectedIndex: nextSelectedIndex,
        }));
        return;
      }

      // Clear number buffer when executing other commands
      numberBufferRef.current = "";

      let newIndex = currentState.selectedIndex;
      let indexChanged = false;

      if (key === "j" || key === "\x1b[B") {
        if (currentState.todos.length === 0) {
          return;
        }
        newIndex = Math.min(
          currentState.selectedIndex + 1,
          currentState.todos.length - 1,
        );
        indexChanged = true;
      } else if (key === "k" || key === "\x1b[A") {
        if (currentState.todos.length === 0) {
          return;
        }
        newIndex = Math.max(currentState.selectedIndex - 1, 0);
        indexChanged = true;
      } else if (key === "\r" || key === "\n" || key === " ") {
        if (currentState.todos.length === 0) {
          return;
        }

        // Save state for undo (deep copy todos array)
        historyRef.current = {
          todos: currentState.todos.map((todo) => ({ ...todo })),
          lines: [...currentState.lines],
        };

        const updatedTodos = [...currentState.todos];
        updatedTodos[currentState.selectedIndex].checked =
          !updatedTodos[currentState.selectedIndex].checked;

        pendingChangeRef.current = true;

        setState((prev) => ({
          ...prev,
          todos: updatedTodos,
        }));
        return;
      } else {
        return;
      }

      if (indexChanged && newIndex !== currentState.selectedIndex) {
        setState((prev) => ({
          ...prev,
          selectedIndex: newIndex,
        }));
      }
    };

    // Enable raw mode
    if (process.stdin.isTTY) {
      process.stdin.setRawMode(true);
    }
    process.stdin.resume();
    process.stdin.setEncoding("utf-8");

    const dataListener = (chunk: Buffer) => {
      handleKey(chunk.toString());
    };

    process.stdin.on("data", dataListener);

    return () => {
      if (process.stdin.isTTY) {
        process.stdin.setRawMode(false);
      }
      process.stdin.pause();
      process.stdin.removeListener("data", dataListener);
    };
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
  } = state;

  if (isLoading) {
    return <Text>Loading todos...</Text>;
  }

  if (error) {
    return <Text color="red">Error: {error}</Text>;
  }

  if (showHelp) {
    // Calculate column widths based on longest shortcut
    const navShortcuts = ["j / ↓", "k / ↑", "5j / 3k"];
    const editShortcuts = ["space", "n", "e", "d"];
    const otherShortcuts = ["u", "?", "q"];

    const navWidth = Math.max(...navShortcuts.map((s) => s.length)) + 1;
    const editWidth = Math.max(...editShortcuts.map((s) => s.length)) + 1;
    const otherWidth = Math.max(...otherShortcuts.map((s) => s.length)) + 1;

    const navColWidth = navWidth + 12; // Add space for description alignment
    const editColWidth = editWidth + 12;
    const otherColWidth = otherWidth + 8;

    const totalWidth = navColWidth + editColWidth + otherColWidth;

    const centerText = (text: string, width: number): string => {
      const padding = Math.max(0, width - text.length);
      const leftPad = Math.floor(padding / 2);
      const rightPad = padding - leftPad;
      return " ".repeat(leftPad) + text + " ".repeat(rightPad);
    };

    return (
      <Box flexDirection="column" paddingX={2} paddingY={1}>
        <Text></Text>
        <Box>
          <Box width={navColWidth}>
            <Text bold color="cyan">
              {centerText("NAVIGATION", navColWidth)}
            </Text>
          </Box>
          <Box width={editColWidth}>
            <Text bold color="cyan">
              {centerText("EDITING", editColWidth)}
            </Text>
          </Box>
          <Box width={otherColWidth}>
            <Text bold color="cyan">
              {centerText("OTHER", otherColWidth)}
            </Text>
          </Box>
        </Box>
        <Text color="gray">{"─".repeat(totalWidth)}</Text>
        <Box>
          <Box flexDirection="column" width={navColWidth}>
            <Box width={navColWidth}>
              <Box width={navWidth}>
                <Text>
                  <Text color="cyan">j</Text>
                  <Text> / ↓</Text>
                </Text>
              </Box>
              <Text>Move down</Text>
            </Box>
            <Box width={navColWidth}>
              <Box width={navWidth}>
                <Text>
                  <Text color="cyan">k</Text>
                  <Text> / ↑</Text>
                </Text>
              </Box>
              <Text>Move up</Text>
            </Box>
            <Box width={navColWidth}>
              <Box width={navWidth}>
                <Text>
                  <Text color="cyan">5j</Text>
                  <Text> / </Text>
                  <Text color="cyan">3k</Text>
                </Text>
              </Box>
              <Text>Jump by count</Text>
            </Box>
          </Box>
          <Box flexDirection="column" width={editColWidth}>
            <Box width={editColWidth}>
              <Box width={editWidth}>
                <Text color="cyan">space</Text>
              </Box>
              <Text>Toggle</Text>
            </Box>
            <Box width={editColWidth}>
              <Box width={editWidth}>
                <Text color="cyan">n</Text>
              </Box>
              <Text>New todo</Text>
            </Box>
            <Box width={editColWidth}>
              <Box width={editWidth}>
                <Text color="cyan">e</Text>
              </Box>
              <Text>Edit text</Text>
            </Box>
            <Box width={editColWidth}>
              <Box width={editWidth}>
                <Text color="cyan">d</Text>
              </Box>
              <Text>Delete</Text>
            </Box>
          </Box>
          <Box flexDirection="column" width={otherColWidth}>
            <Box width={otherColWidth}>
              <Box width={otherWidth}>
                <Text color="cyan">u</Text>
              </Box>
              <Text>Undo</Text>
            </Box>
            <Box width={otherColWidth}>
              <Box width={otherWidth}>
                <Text color="cyan">?</Text>
              </Box>
              <Text>Help</Text>
            </Box>
            <Box width={otherColWidth}>
              <Box width={otherWidth}>
                <Text color="cyan">q</Text>
              </Box>
              <Text>Quit</Text>
            </Box>
          </Box>
        </Box>
        <Text color="gray">{"─".repeat(totalWidth)}</Text>
        <Text color="gray">Press ? or q to exit</Text>
        <Text></Text>
      </Box>
    );
  }

  if (inputMode) {
    return (
      <Box flexDirection="column">
        <Box>
          <Text color="cyan">➜ </Text>
          <Text color="green">[ ] </Text>
          <Text>{displayInputBuffer}</Text>
          <Text color="gray">_</Text>
        </Box>
        <Text color="gray">(Press Enter to confirm, Esc to cancel)</Text>
      </Box>
    );
  }

  if (editMode) {
    return (
      <Box flexDirection="column">
        <Box>
          <Text color="cyan">➜ </Text>
          <Text color="green">[ ] </Text>
          <Text>{displayInputBuffer}</Text>
          <Text color="gray">_</Text>
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
        const arrow = isSelected ? chalk.cyan("➜") : " ";
        const text = isSelected ? chalk.bold.whiteBright(todo.text) : todo.text;
        const relativeIndex = index - selectedIndex;
        let relativeDisplay = "";
        if (relativeIndex === 0) {
          relativeDisplay = "";
        } else if (relativeIndex > 0) {
          relativeDisplay = `+${relativeIndex}`;
        } else {
          relativeDisplay = `${relativeIndex}`;
        }

        return (
          <Box key={todo.lineNo}>
            <Text color="gray" dimColor>
              {relativeDisplay.padEnd(3)}
            </Text>
            <Text>{arrow}</Text>
            <Text> </Text>
            <Text>{checkbox}</Text>
            <Text> </Text>
            <Text>{text}</Text>
          </Box>
        );
      })}
      <Box marginTop={1}>
        <Text color="gray">
          <Text color="cyan">?</Text>
          {": help  |  "}
          <Text color="cyan">j</Text>/<Text color="cyan">k</Text>
          {": nav  |  "}
          <Text color="cyan">space</Text>
          {": toggle  |  "}
          <Text color="cyan">q</Text>
          {": quit"}
        </Text>
      </Box>
    </Box>
  );
}
