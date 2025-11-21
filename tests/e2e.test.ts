import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { execSync } from "child_process";
import fs from "fs";
import path from "path";

const TEST_FILE = "/tmp/tdx-test-" + process.pid + ".md";
const CLI = "bun run src/cli.ts";

// Helper to run CLI command
function run(args: string = ""): string {
  try {
    return execSync(`${CLI} ${TEST_FILE} ${args}`, {
      cwd: path.join(__dirname, ".."),
      encoding: "utf-8",
      timeout: 10000,
    }).trim();
  } catch (error: any) {
    return error.stdout?.toString() || error.message;
  }
}

// Helper to run piped input to TUI
function runPiped(input: string, timeoutMs: number = 8000): string {
  try {
    // Use gtimeout on macOS if available, otherwise fall back to perl-based timeout
    const result = execSync(`printf "${input}" | ${CLI} ${TEST_FILE}`, {
      cwd: path.join(__dirname, ".."),
      encoding: "utf-8",
      shell: "/bin/bash",
      timeout: timeoutMs,
    });
    return result.trim();
  } catch (error: any) {
    // Timeout or other error - return whatever output we got
    return error.stdout?.toString().trim() || "";
  }
}

// Helper to read test file
function readTestFile(): string {
  try {
    return fs.readFileSync(TEST_FILE, "utf-8");
  } catch {
    return "";
  }
}

// Helper to get todos from file
function getTodos(): string[] {
  const content = readTestFile();
  const matches = content.match(/^- \[[ x]\] .+$/gm);
  return matches || [];
}

describe("E2E Tests", () => {
  beforeEach(() => {
    // Clean up test file before each test
    try {
      fs.unlinkSync(TEST_FILE);
    } catch {
      // File doesn't exist, that's fine
    }
  });

  afterEach(() => {
    // Clean up test file after each test
    try {
      fs.unlinkSync(TEST_FILE);
    } catch {
      // File doesn't exist, that's fine
    }
  });

  describe("CLI Commands", () => {
    it("should create file and add todo via CLI", () => {
      const output = run('add "First todo"');
      expect(output).toContain("Added: First todo");

      const todos = getTodos();
      expect(todos).toHaveLength(1);
      expect(todos[0]).toBe("- [ ] First todo");
    });

    it("should list todos", () => {
      run('add "Todo 1"');
      run('add "Todo 2"');

      const output = run("list");
      expect(output).toContain("1. [ ] Todo 1");
      expect(output).toContain("2. [ ] Todo 2");
    });

    it("should toggle todo", () => {
      run('add "Toggle me"');

      let todos = getTodos();
      expect(todos[0]).toBe("- [ ] Toggle me");

      run("toggle 1");

      todos = getTodos();
      expect(todos[0]).toBe("- [x] Toggle me");

      // Toggle back
      run("toggle 1");

      todos = getTodos();
      expect(todos[0]).toBe("- [ ] Toggle me");
    });

    it("should edit todo", () => {
      run('add "Original text"');
      run('edit 1 "Edited text"');

      const todos = getTodos();
      expect(todos[0]).toBe("- [ ] Edited text");
    });

    it("should delete todo", () => {
      run('add "Keep me"');
      run('add "Delete me"');
      run('add "Keep me too"');

      run("delete 2");

      const todos = getTodos();
      expect(todos).toHaveLength(2);
      expect(todos[0]).toBe("- [ ] Keep me");
      expect(todos[1]).toBe("- [ ] Keep me too");
    });

    it("should show help", () => {
      const output = execSync(`${CLI} help`, {
        cwd: path.join(__dirname, ".."),
        encoding: "utf-8",
      });
      expect(output).toContain("tdx - A fast, lightweight todo manager");
      expect(output).toContain("Usage:");
    });
  });

  describe("TUI Piped Input", () => {
    it("should create new todo via TUI", () => {
      runPiped("n\\rNew TUI todo\\r");

      const todos = getTodos();
      expect(todos).toHaveLength(1);
      expect(todos[0]).toBe("- [ ] New TUI todo");
    });

    it("should create multiple todos via TUI", () => {
      runPiped("n\\rFirst\\rn\\rSecond\\rn\\rThird\\r");

      const todos = getTodos();
      expect(todos).toHaveLength(3);
      expect(todos[0]).toBe("- [ ] First");
      expect(todos[1]).toBe("- [ ] Second");
      expect(todos[2]).toBe("- [ ] Third");
    });

    it("should toggle todo via TUI (space)", () => {
      run('add "Toggle via space"');

      runPiped(" ");

      const todos = getTodos();
      expect(todos[0]).toBe("- [x] Toggle via space");
    });

    it("should toggle todo via TUI (enter)", () => {
      run('add "Toggle via enter"');

      runPiped("\\r");

      const todos = getTodos();
      expect(todos[0]).toBe("- [x] Toggle via enter");
    });

    it("should delete todo via TUI", () => {
      run('add "Delete me"');
      run('add "Keep me"');

      runPiped("d");

      const todos = getTodos();
      expect(todos).toHaveLength(1);
      expect(todos[0]).toBe("- [ ] Keep me");
    });

    it("should navigate down and toggle second todo", () => {
      run('add "First"');
      run('add "Second"');
      run('add "Third"');

      // Navigate down once and toggle
      runPiped("j ");

      const todos = getTodos();
      expect(todos[0]).toBe("- [ ] First");
      expect(todos[1]).toBe("- [x] Second");
      expect(todos[2]).toBe("- [ ] Third");
    });

    it("should navigate with count (5j)", () => {
      // Add 10 todos
      for (let i = 1; i <= 10; i++) {
        run(`add "Todo ${i}"`);
      }

      // Navigate down 5 and toggle
      runPiped("5j ");

      const todos = getTodos();
      // Index 5 (6th todo) should be toggled
      expect(todos[5]).toBe("- [x] Todo 6");
    });

    it("should edit existing todo via TUI", () => {
      run('add "Original"');

      // Press 'e' to edit, clear with backspaces, type new text
      // Note: We need to send backspaces to clear "Original"
      const backspaces = "\\x7f".repeat(8); // 8 backspaces
      runPiped(`e${backspaces}Edited\\r`);

      const todos = getTodos();
      expect(todos[0]).toBe("- [ ] Edited");
    });

    it("should undo last action", () => {
      run('add "First"');
      run('add "Second"');

      // Delete first, then undo
      runPiped("du");

      const todos = getTodos();
      expect(todos).toHaveLength(2);
      expect(todos[0]).toBe("- [ ] First");
    });

    it("should handle empty input gracefully", () => {
      // Just pressing enter with no text should not create todo
      runPiped("n\\r\\r");

      const todos = getTodos();
      // With piped input, empty enter stays in input mode
      // so second \\r also does nothing
      expect(todos).toHaveLength(0);
    });

    it("should show help menu with ?", () => {
      run('add "Test"');

      const output = runPiped("?");
      expect(output).toContain("NAVIGATION");
      expect(output).toContain("EDITING");
    });
  });

  describe("Edge Cases", () => {
    it("should handle special characters in todo text", () => {
      run('add "Todo with special chars: @#$%^&*()"');

      const todos = getTodos();
      expect(todos[0]).toContain("special chars");
    });

    it("should handle very long todo text", () => {
      const longText = "A".repeat(200);
      run(`add "${longText}"`);

      const todos = getTodos();
      expect(todos[0]).toContain(longText);
    });

    it("should handle multiple toggles", () => {
      run('add "Toggle multiple times"');

      // Toggle 3 times
      runPiped("   ");

      const todos = getTodos();
      // Should end up checked (odd number of toggles)
      expect(todos[0]).toBe("- [x] Toggle multiple times");
    });

    it("should preserve file header", () => {
      run('add "Test"');

      const content = readTestFile();
      expect(content.startsWith("# Todos")).toBe(true);
    });

    it("should handle delete on empty list gracefully", () => {
      // Should not crash
      runPiped("d");

      const todos = getTodos();
      expect(todos).toHaveLength(0);
    });

    it("should handle navigation on empty list", () => {
      // Should not crash
      runPiped("jjkk");

      const todos = getTodos();
      expect(todos).toHaveLength(0);
    });
  });
});
