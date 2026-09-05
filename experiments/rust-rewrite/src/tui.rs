use crate::editor::Editor;
use crossterm::{
    event::{
        self, DisableBracketedPaste, EnableBracketedPaste, Event, KeyCode, KeyEventKind,
        KeyModifiers,
    },
    execute,
};
use ratatui::{
    Frame,
    layout::{Constraint, Layout},
    style::{Color, Modifier, Style},
    widgets::{Block, List, ListItem, ListState, Paragraph},
};
use std::{io, path::Path};

struct Input {
    op: &'static str,
    text: String,
}
struct App {
    editor: Editor,
    selection: ListState,
    input: Option<Input>,
    status: String,
    confirm_quit: bool,
    confirm_reload: bool,
}
impl App {
    fn selected(&self) -> usize {
        self.selection.selected().unwrap_or(0)
    }
    fn clamp(&mut self) {
        let len = self.editor.doc.tasks.len();
        self.selection.select(if len == 0 {
            None
        } else {
            Some(self.selected().min(len - 1))
        });
    }
    fn result(&mut self, result: Result<(), String>) {
        self.status = result.err().unwrap_or_else(|| "Saved".into());
        self.clamp();
    }
    fn draw(&mut self, frame: &mut Frame, path: &Path) {
        let areas = Layout::vertical([
            Constraint::Length(2),
            Constraint::Min(1),
            Constraint::Length(3),
        ])
        .split(frame.area());
        let readonly = self.editor.readonly || self.editor.doc.readonly;
        let title = format!(
            "tdx · Rust prototype   {}{}{}",
            path.display(),
            if readonly { "  READ ONLY" } else { "" },
            if self.editor.dirty { "  UNSAVED" } else { "" }
        );
        frame.render_widget(
            Paragraph::new(title).style(Style::default().fg(Color::Cyan)),
            areas[0],
        );
        let items: Vec<_> = self
            .editor
            .doc
            .tasks
            .iter()
            .map(|task| {
                ListItem::new(format!(
                    "{}[{}] {}",
                    "  ".repeat(task.depth),
                    if task.checked { "x" } else { " " },
                    display_text(&task.text)
                ))
            })
            .collect();
        let list = List::new(items)
            .block(Block::bordered().title(format!(" {} tasks ", self.editor.doc.tasks.len())))
            .highlight_style(
                Style::default()
                    .fg(Color::Cyan)
                    .add_modifier(Modifier::BOLD),
            )
            .highlight_symbol("› ");
        frame.render_stateful_widget(list, areas[1], &mut self.selection);
        let footer = if let Some(input) = &self.input {
            format!(
                "{}: {}▏\nEnter save · Esc cancel · Ctrl-U clear",
                input.op.to_uppercase(),
                display_text(&input.text)
            )
        } else {
            format!(
                "j/k move · space toggle · a/e/d edit · u undo · r reload · q quit\n{}",
                self.status
            )
        };
        frame.render_widget(Paragraph::new(footer), areas[2]);
    }
    fn handle(&mut self, event: Event) -> bool {
        if let Event::Paste(text) = &event {
            if let Some(input) = &mut self.input {
                // Keep the one-line edit contract; ignore pasted control bytes.
                input.text.extend(text.chars().filter(|c| !c.is_control()));
            }
            return false;
        }
        let Event::Key(key) = event else { return false };
        if key.kind == KeyEventKind::Release {
            return false;
        }
        if key.modifiers.contains(KeyModifiers::CONTROL)
            && matches!(key.code, KeyCode::Char('c' | 'd'))
        {
            if self.editor.dirty && !self.confirm_quit {
                self.confirm_quit = true;
                self.status =
                    "Unsaved changes. Press quit again to discard, or r to reload.".into();
                return false;
            }
            return true;
        }
        if let Some(input) = &mut self.input {
            match key.code {
                KeyCode::Esc => {
                    self.input = None;
                    self.status = "Cancelled".into();
                }
                KeyCode::Enter => {
                    let input = self.input.take().unwrap();
                    if input.text.trim().is_empty() {
                        self.status = "Cancelled empty input".into();
                    } else {
                        let result = self.editor.apply(input.op, self.selected(), &input.text);
                        self.result(result);
                    }
                }
                KeyCode::Backspace => {
                    input.text.pop();
                }
                KeyCode::Char('u') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                    input.text.clear()
                }
                KeyCode::Char(c)
                    if !key
                        .modifiers
                        .intersects(KeyModifiers::CONTROL | KeyModifiers::ALT) =>
                {
                    input.text.push(c)
                }
                _ => {}
            }
            return false;
        }
        if key.code == KeyCode::Char('q') {
            if !self.editor.dirty || self.confirm_quit {
                return true;
            }
            self.confirm_quit = true;
            self.status = "Unsaved changes. Press q again to discard, or r to reload.".into();
            return false;
        }
        self.confirm_quit = false;
        if key.code != KeyCode::Char('r') {
            self.confirm_reload = false;
        }
        match key.code {
            KeyCode::Down | KeyCode::Char('j') => {
                self.selection.select(Some(self.selected() + 1));
                self.clamp();
            }
            KeyCode::Up | KeyCode::Char('k') => {
                self.selection
                    .select(Some(self.selected().saturating_sub(1)));
                self.clamp();
            }
            KeyCode::Char(' ') | KeyCode::Char('d') => {
                let result = self.editor.apply(
                    if key.code == KeyCode::Char(' ') {
                        "toggle"
                    } else {
                        "delete"
                    },
                    self.selected(),
                    "",
                );
                self.result(result);
            }
            KeyCode::Char('a' | 'e') => {
                if self.editor.readonly || self.editor.doc.readonly {
                    self.status = "read-only: editing is disabled".into();
                } else if key.code == KeyCode::Char('a') {
                    self.input = Some(Input {
                        op: "add",
                        text: String::new(),
                    });
                } else if let Some(task) = self.editor.doc.tasks.get(self.selected()) {
                    self.input = Some(Input {
                        op: "edit",
                        text: task.text.clone(),
                    });
                }
            }
            KeyCode::Char('u') => {
                let result = self.editor.undo();
                self.result(result);
            }
            KeyCode::Char('r') => {
                if self.editor.dirty && !self.confirm_reload {
                    self.confirm_reload = true;
                    self.status = "Unsaved changes. Press r again to discard and reload.".into();
                } else {
                    let result = self.editor.reload();
                    self.result(result);
                    self.confirm_reload = false;
                }
            }
            _ => {}
        }
        false
    }
}
fn display_text(text: &str) -> String {
    text.chars()
        .map(|c| if c.is_control() { '�' } else { c })
        .collect()
}
struct PasteGuard;
impl Drop for PasteGuard {
    fn drop(&mut self) {
        let _ = execute!(io::stdout(), DisableBracketedPaste);
    }
}

pub fn run(editor: Editor, path: &Path) -> io::Result<()> {
    // Ratatui's run restores raw mode/alternate screen on return and panic.
    ratatui::run(|terminal| {
        execute!(io::stdout(), EnableBracketedPaste)?;
        let _paste = PasteGuard;
        let mut app = App {
            editor,
            selection: ListState::default(),
            input: None,
            status: "Prototype · no version history · r reloads external changes".into(),
            confirm_quit: false,
            confirm_reload: false,
        };
        app.clamp();
        loop {
            terminal.draw(|frame| app.draw(frame, path))?;
            if app.handle(event::read()?) {
                return Ok(());
            }
        }
    })
}
