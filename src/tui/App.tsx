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
  });

  // Use refs to track mutable state
  const pendingChangeRef = useRef<boolean>(false);
  const inputBufferRef = useRef<string>("");
  const stateRef = useRef(state);
  const historyRef = useRef<{ todos: Todo[]; lines: string[] } | null>(null);

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
  } = state;

  if (isLoading) {
    return <Text>Loading todos...</Text>;
  }

  if (error) {
    return <Text color="red">Error: {error}</Text>;
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

        return (
          <Box key={todo.lineNo}>
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
          <Text color="cyan">j</Text>/<Text color="cyan">k</Text>
          {" nav  |  "}
          <Text color="cyan">space</Text>
          {": toggle  |  "}
          <Text color="cyan">n</Text>
          {": new  |  "}
          <Text color="cyan">d</Text>
          {": delete  |  "}
          <Text color="cyan">e</Text>
          {": edit  |  "}
          <Text color="cyan">u</Text>
          {": undo  |  "}
          <Text color="cyan">q</Text>
          {": quit"}
        </Text>
      </Box>
    </Box>
  );
}
