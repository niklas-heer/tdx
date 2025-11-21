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
}

export default function App() {
  const [state, setState] = useState<AppState>({
    todos: [],
    lines: [],
    selectedIndex: 0,
    isLoading: true,
    error: null,
  });

  // Use ref to track if we should persist changes
  const pendingChangeRef = useRef<boolean>(false);

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
      // Handle special characters and sequences
      if (key === "\u001b") {
        // Escape key
        process.exit(0);
        return;
      }

      if (key === "q" || key === "Q") {
        process.exit(0);
        return;
      }

      if (key === "\u0003") {
        // Ctrl+C
        process.exit(0);
        return;
      }

      setState((prev) => {
        if (key === "d") {
          if (prev.todos.length === 0) {
            return prev;
          }

          const deletedTodo = prev.todos[prev.selectedIndex];
          const remainingTodos = prev.todos
            .filter((_, idx) => idx !== prev.selectedIndex)
            .map((todo, idx) => ({
              ...todo,
              index: idx + 1,
              lineNo:
                todo.lineNo > deletedTodo.lineNo
                  ? todo.lineNo - 1
                  : todo.lineNo,
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
        let newIndex = prev.selectedIndex;

        if (key === "j" || key === "\x1b[B") {
          if (prev.todos.length === 0) {
            return prev;
          }
          newIndex = Math.min(prev.selectedIndex + 1, prev.todos.length - 1);
        } else if (key === "k" || key === "\x1b[A") {
          if (prev.todos.length === 0) {
            return prev;
          }
          newIndex = Math.max(prev.selectedIndex - 1, 0);
        } else if (key === "\r" || key === "\n" || key === " ") {
          if (prev.todos.length === 0) {
            return prev;
          }

          const updatedTodos = [...prev.todos];
          updatedTodos[prev.selectedIndex].checked =
            !updatedTodos[prev.selectedIndex].checked;

          pendingChangeRef.current = true;

          return {
            ...prev,
            todos: updatedTodos,
          };
        } else {
          return prev;
        }

        if (newIndex === prev.selectedIndex) {
          return prev;
        }

        return {
          ...prev,
          selectedIndex: newIndex,
        };
      });
    };

    // Enable raw mode
    if (process.stdin.isTTY) {
      process.stdin.setRawMode(true);
    }
    process.stdin.resume();
    process.stdin.setEncoding("utf-8");

    process.stdin.on("data", (chunk) => {
      handleKey(chunk.toString());
    });

    return () => {
      if (process.stdin.isTTY) {
        process.stdin.setRawMode(false);
      }
      process.stdin.pause();
    };
  }, [state.isLoading, state.error, state.todos.length]);

  const { todos, selectedIndex, isLoading, error } = state;

  if (isLoading) {
    return <Text>Loading todos...</Text>;
  }

  if (error) {
    return <Text color="red">Error: {error}</Text>;
  }

  if (todos.length === 0) {
    return (
      <Box flexDirection="column">
        <Text color="gray">
          No todos yet. Use "tdx add &lt;text&gt;" to create one.
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
    </Box>
  );
}
