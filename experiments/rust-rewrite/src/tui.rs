use crate::{editor::Editor, history::Version};
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
use std::{
    io,
    path::Path,
    time::{Duration, Instant},
};

struct Input {
    op: &'static str,
    text: String,
}
struct Browser {
    versions: Vec<Version>,
    selection: ListState,
    preview: String,
    scroll: u16,
    confirm: bool,
}
struct App<'a> {
    editor: &'a mut Editor,
    browser: Option<Browser>,
    selection: ListState,
    input: Option<Input>,
    status: String,
    confirm_quit: bool,
    confirm_reload: bool,
}
impl App<'_> {
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
        self.status = result
            .err()
            .map(|e| {
                if e.contains("externally") {
                    format!("{e}; :reload or :force-save")
                } else {
                    e
                }
            })
            .unwrap_or_else(|| "Saved".into());
        self.clamp();
    }
    fn open_versions(&mut self) -> Result<(), String> {
        let versions = self.editor.store.versions()?;
        let preview = if let Some(version) = versions.first() {
            self.editor.store.version(version.id)?
        } else {
            "No versions available".into()
        };
        let mut selection = ListState::default();
        if !versions.is_empty() {
            selection.select(Some(0));
        }
        self.browser = Some(Browser {
            versions,
            selection,
            preview,
            scroll: 0,
            confirm: false,
        });
        Ok(())
    }
    fn command(&mut self, text: &str) {
        let result = match text.trim() {
            "versions" => self.open_versions(),
            "reload" => self.editor.reload(),
            "force-save" => self.editor.force_save(),
            _ => Err("unknown command; use versions, reload or force-save".into()),
        };
        self.result(result);
    }
    fn check_disk(&mut self) -> bool {
        if self.input.is_some() || self.browser.is_some() || self.editor.dirty {
            return false;
        }
        match self.editor.store.changed() {
            Ok(true) => {
                let result = self.editor.reload();
                self.result(result);
                true
            }
            Err(error) => {
                self.status = error;
                true
            }
            Ok(false) => false,
        }
    }
    fn draw_versions(&mut self, frame: &mut Frame) {
        let browser = self.browser.as_mut().unwrap();
        let areas = Layout::vertical([
            Constraint::Length(2),
            Constraint::Min(1),
            Constraint::Length(3),
        ])
        .split(frame.area());
        frame.render_widget(
            Paragraph::new("FILE VERSION HISTORY").style(Style::default().fg(Color::Cyan)),
            areas[0],
        );
        let panes = Layout::horizontal([Constraint::Percentage(25), Constraint::Percentage(75)])
            .split(areas[1]);
        let rows: Vec<_> = browser
            .versions
            .iter()
            .map(|v| ListItem::new(format!("#{} {}", v.id, v.created_at)))
            .collect();
        let list = List::new(rows)
            .block(Block::bordered().title("Versions"))
            .highlight_symbol("› ")
            .highlight_style(Style::default().fg(Color::Cyan));
        frame.render_stateful_widget(list, panes[0], &mut browser.selection);
        let preview = browser
            .preview
            .lines()
            .map(display_text)
            .collect::<Vec<_>>()
            .join("\n");
        frame.render_widget(
            Paragraph::new(preview)
                .block(Block::bordered().title("Saved snapshot"))
                .scroll((browser.scroll, 0)),
            panes[1],
        );
        let footer = if browser.confirm {
            format!(
                "Restore version #{}? [y/N]\n{}",
                browser.versions[browser.selection.selected().unwrap()].id,
                self.status
            )
        } else {
            format!(
                "[↑/↓] Navigate  [PgUp/PgDn] Scroll  [Enter] Restore  [Esc] Close\n{}",
                self.status
            )
        };
        frame.render_widget(Paragraph::new(footer), areas[2]);
    }
    fn browser_key(&mut self, key: KeyCode) {
        let browser = self.browser.as_mut().unwrap();
        if browser.confirm {
            browser.confirm = false;
            if matches!(key, KeyCode::Char('y' | 'Y')) {
                let id = browser.versions[browser.selection.selected().unwrap()].id;
                let result = self.editor.restore(id);
                if result.is_ok()
                    || result
                        .as_ref()
                        .is_err_and(|e| e.starts_with("file saved, but"))
                {
                    self.browser = None;
                }
                self.result(result);
            }
            return;
        }
        let current = browser.selection.selected().unwrap_or(0);
        let next = match key {
            KeyCode::Esc | KeyCode::Char('q') => {
                self.browser = None;
                return;
            }
            KeyCode::Enter if !browser.versions.is_empty() => {
                browser.confirm = true;
                return;
            }
            KeyCode::Down | KeyCode::Char('j') => {
                (current + 1).min(browser.versions.len().saturating_sub(1))
            }
            KeyCode::Up | KeyCode::Char('k') => current.saturating_sub(1),
            KeyCode::PageDown => {
                browser.scroll = browser.scroll.saturating_add(10).min(
                    browser
                        .preview
                        .lines()
                        .count()
                        .saturating_sub(1)
                        .min(u16::MAX as usize) as u16,
                );
                return;
            }
            KeyCode::PageUp => {
                browser.scroll = browser.scroll.saturating_sub(10);
                return;
            }
            _ => return,
        };
        if let Some(version) = browser.versions.get(next) {
            match self.editor.store.version(version.id) {
                Ok(preview) => {
                    browser.preview = preview;
                    browser.selection.select(Some(next));
                    browser.scroll = 0;
                }
                Err(error) => self.status = error,
            }
        }
    }
    fn draw(&mut self, frame: &mut Frame, path: &Path) {
        if self.browser.is_some() {
            self.draw_versions(frame);
            return;
        }
        let areas = Layout::vertical([
            Constraint::Length(2),
            Constraint::Min(1),
            Constraint::Length(3),
        ])
        .split(frame.area());
        let readonly = self.editor.readonly || self.editor.doc.readonly;
        let title = format!(
            "tdx · Rust prototype{}{}   {}",
            if readonly { "  READ ONLY" } else { "" },
            if self.editor.dirty { "  UNSAVED" } else { "" },
            path.display()
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
                "j/k move · space toggle · a/e/d edit · u undo · r reload · v history · q quit\n{}",
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
        if self.browser.is_some() {
            self.browser_key(key.code);
            return false;
        }
        if let Some(input) = &mut self.input {
            match key.code {
                KeyCode::Esc => {
                    self.input = None;
                    self.status = "Cancelled".into();
                }
                KeyCode::Enter => {
                    let input = self.input.take().unwrap();
                    if input.op == "command" {
                        self.command(&input.text);
                    } else if input.text.trim().is_empty() {
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
            KeyCode::Char(':') => {
                self.input = Some(Input {
                    op: "command",
                    text: String::new(),
                });
            }
            KeyCode::Char('v') => {
                let result = self.open_versions();
                self.result(result);
            }
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

pub fn run(editor: &mut Editor, path: &Path) -> io::Result<()> {
    // Ratatui's run restores raw mode/alternate screen on return and panic.
    ratatui::run(|terminal| {
        execute!(io::stdout(), EnableBracketedPaste)?;
        let _paste = PasteGuard;
        let mut app = App {
            editor,
            browser: None,
            selection: ListState::default(),
            input: None,
            status: "Version history enabled · v browse · :force-save resolves conflicts".into(),
            confirm_quit: false,
            confirm_reload: false,
        };
        app.clamp();
        let mut checked = Instant::now();
        let mut redraw = true;
        loop {
            if redraw {
                terminal.draw(|frame| app.draw(frame, path))?;
            }
            redraw = false;
            if event::poll(Duration::from_millis(100))? {
                if app.handle(event::read()?) {
                    return Ok(());
                }
                redraw = true;
            }
            if checked.elapsed() >= Duration::from_millis(500) {
                redraw |= app.check_disk();
                checked = Instant::now();
            }
        }
    })
}
