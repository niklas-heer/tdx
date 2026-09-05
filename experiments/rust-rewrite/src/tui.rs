use crate::{
    actions::Action,
    clipboard,
    config::{self, Config, Overrides, Settings},
    document::Document,
    editor::Editor,
    history::Version,
    input::{self, Buffer},
    presentation::{self, Links},
    recent::{Recent, RecentFile},
    store::Store,
};
use chrono::{Local, NaiveDate};
use crossterm::{
    event::{
        DisableBracketedPaste, EnableBracketedPaste, Event, KeyCode, KeyEvent, KeyEventKind,
        KeyModifiers,
    },
    execute,
};
use ratatui::{
    Frame,
    layout::{Constraint, Layout, Rect},
    style::{Color, Modifier, Style},
    text::{Line, Span},
    widgets::{Block, BorderType, Clear, List, ListItem, ListState, Paragraph, Wrap},
};
use similar::{ChangeTag, TextDiff};
use std::{
    collections::{BTreeMap, BTreeSet},
    io,
    path::{Path, PathBuf},
    time::{Duration, Instant},
};
use unicode_width::{UnicodeWidthChar, UnicodeWidthStr};

const COMMANDS: &[(&str, &str)] = &[
    ("check-all", "Mark all tasks complete"),
    ("uncheck-all", "Mark all tasks incomplete"),
    ("sort-done", "Sort incomplete tasks first"),
    ("sort-due", "Sort by earliest due date"),
    ("sort-priority", "Sort highest priority first"),
    ("filter-done", "Toggle completed task filter"),
    ("filter-due", "Toggle tasks with due dates"),
    ("filter-overdue", "Toggle overdue tasks"),
    ("filter-today", "Toggle tasks due today"),
    ("filter-week", "Toggle tasks due within seven days"),
    ("clear-done", "Delete completed tasks"),
    ("read-only", "Toggle automatic saves"),
    ("save", "Save current document"),
    ("wrap", "Toggle word wrap"),
    ("line-numbers", "Toggle relative line numbers"),
    ("set-max-visible", "Set visible item limit"),
    ("sections", "Browse and edit sections"),
    ("all-sections", "Clear section focus and folds"),
    ("show-headings", "Toggle heading display"),
    ("reload", "Reload and discard local changes"),
    ("force-save", "Capture disk revision and overwrite"),
    ("diff", "Compare local and disk changes"),
    ("theme", "Choose theme with live preview"),
    ("versions", "Browse and restore saved versions"),
];
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum Mode {
    Normal,
    Input,
    Search,
    Commands,
    Tags,
    Priorities,
    Due,
    Sections,
    Heading,
    Recent,
    Theme,
    Versions,
    Diff,
    Help,
    MaxVisible,
    Move,
}
struct App<'a> {
    editor: &'a mut Editor,
    path: PathBuf,
    config: Config,
    flags: Overrides,
    settings: Settings,
    mode: Mode,
    selected: usize,
    cursor: usize,
    input: Buffer,
    action: Action,
    status: String,
    tags: BTreeSet<String>,
    priorities: BTreeSet<i64>,
    due: String,
    section: Option<usize>,
    folded: BTreeSet<usize>,
    line_numbers: bool,
    number: String,
    g: bool,
    versions: Vec<Version>,
    preview: String,
    diff: Vec<Line<'static>>,
    scroll: usize,
    confirm: bool,
    recent: Vec<RecentFile>,
    themes: BTreeMap<String, config::Colors>,
    theme: String,
    original_theme: String,
    moving: Option<(Document, usize, bool)>,
    quit_confirm: bool,
    links: Links,
}
impl<'a> App<'a> {
    fn new(editor: &'a mut Editor, path: &Path, config: Config, flags: Overrides) -> Self {
        let settings = Settings::new(&config, &editor.doc.metadata, &flags);
        editor.readonly = settings.read_only;
        let selected = config::directory()
            .ok()
            .and_then(|dir| Recent::load(&dir, config.recent.max_files).ok())
            .and_then(|r| r.cursor(path))
            .unwrap_or(0);
        let theme = config.theme.name.clone();
        let themes = config::themes();
        let mut app = Self {
            editor,
            path: path.into(),
            config,
            flags,
            settings,
            mode: Mode::Normal,
            selected,
            cursor: 0,
            input: Buffer::default(),
            action: Action::default(),
            status: String::new(),
            tags: BTreeSet::new(),
            priorities: BTreeSet::new(),
            due: String::new(),
            section: None,
            folded: BTreeSet::new(),
            line_numbers: true,
            number: String::new(),
            g: false,
            versions: vec![],
            preview: String::new(),
            diff: vec![],
            scroll: 0,
            confirm: false,
            recent: vec![],
            themes,
            theme,
            original_theme: String::new(),
            moving: None,
            quit_confirm: false,
            links: Links::new(),
        };
        app.clamp();
        app
    }
    fn color(&self, name: &str) -> Color {
        let colors = self.themes.get(&self.theme).unwrap_or(&self.config.colors);
        let value = colors
            .get(name)
            .filter(|s| !s.is_empty())
            .map(String::as_str)
            .or_else(|| match name {
                "Tag" => colors.get("Warning").map(String::as_str),
                "PriorityLow" | "DueFuture" => colors.get("Dim").map(String::as_str),
                "PriorityHigh" => Some("#f7768e"),
                "PriorityMedium" => Some("#bb9af7"),
                "DueUrgent" => Some("#7dcfff"),
                "DueSoon" => Some("#7aa2f7"),
                _ => None,
            });
        value.and_then(|s| s.parse().ok()).unwrap_or(Color::Reset)
    }

    fn visible(&self) -> Vec<usize> {
        let today = Local::now().date_naive();
        self.editor
            .doc
            .tasks
            .iter()
            .enumerate()
            .filter(|(i, t)| {
                (!self.settings.filter_done || !t.checked)
                    && (self.tags.is_empty()
                        || self
                            .tags
                            .iter()
                            .any(|tag| t.tags.iter().any(|s| s.eq_ignore_ascii_case(tag))))
                    && (self.priorities.is_empty() || self.priorities.contains(&t.priority))
                    && match self.due.as_str() {
                        "" => true,
                        "all" => t.due_date.is_some(),
                        kind => t
                            .due_date
                            .as_ref()
                            .and_then(|s| NaiveDate::parse_from_str(s, "%Y-%m-%d").ok())
                            .is_some_and(|d| match kind {
                                "overdue" => d < today,
                                "today" => d == today,
                                "week" => d >= today && (d - today).num_days() <= 7,
                                _ => true,
                            }),
                    }
                    && self
                        .section
                        .is_none_or(|h| self.editor.doc.section_bounds(h).contains(i))
                    && !self
                        .folded
                        .iter()
                        .any(|h| self.editor.doc.section_bounds(*h).contains(i))
            })
            .map(|(i, _)| i)
            .collect()
    }
    fn clamp(&mut self) {
        let tags = self.tags_available();
        self.tags.retain(|tag| tags.contains(tag));
        let priorities = self.priorities_available();
        self.priorities.retain(|p| priorities.contains(p));
        let visible = self.visible();
        if !visible.contains(&self.selected) {
            self.selected = visible
                .iter()
                .copied()
                .find(|i| *i >= self.selected)
                .or_else(|| visible.last().copied())
                .unwrap_or(0);
        }
    }
    fn best_selection(&self, index: usize, removed: bool) -> usize {
        let tasks = &self.editor.doc.tasks;
        let Some(task) = tasks.get(index) else {
            return 0;
        };
        let visible = self.visible();
        for (i, candidate) in tasks.iter().enumerate().skip(index + 1) {
            if candidate.depth < task.depth {
                break;
            }
            if candidate.depth == task.depth
                && candidate.parent_index == task.parent_index
                && visible.contains(&i)
            {
                return i - usize::from(removed);
            }
        }
        for i in (0..index).rev() {
            if tasks[i].depth < task.depth {
                break;
            }
            if tasks[i].depth == task.depth
                && tasks[i].parent_index == task.parent_index
                && visible.contains(&i)
            {
                return i;
            }
        }
        if let Some(parent) = task.parent_index
            && visible.contains(&(parent - 1))
        {
            return parent - 1;
        }
        visible
            .iter()
            .copied()
            .find(|i| *i > index)
            .map(|i| i - usize::from(removed))
            .or_else(|| visible.iter().copied().rfind(|i| *i < index))
            .unwrap_or(if removed { 0 } else { index })
    }
    fn clear_sections(&mut self) {
        self.section = None;
        self.folded.clear();
        self.clamp();
    }
    fn result(&mut self, result: Result<(), String>) {
        self.status = match result {
            Ok(()) => if self.editor.dirty {
                "Unsaved checklist changes"
            } else {
                "Saved"
            }
            .into(),
            Err(e) => {
                if e.contains("externally") {
                    self.mode = Mode::Diff;
                    self.scroll = 0;
                    self.diff = diff_lines(
                        &std::fs::read_to_string(&self.path).unwrap_or_default(),
                        &self.editor.doc.source,
                    );
                    format!("{e}; :reload or :force-save")
                } else {
                    e
                }
            }
        };
        self.clamp();
    }
    fn apply(&mut self, action: Action) {
        self.editor.readonly = self.settings.read_only;
        let result = self.editor.action(&action);
        if let Ok(index) = &result
            && !matches!(action.kind.as_str(), "rename-heading" | "create-heading")
        {
            self.selected = *index;
        }
        self.result(result.map(|_| ()));
    }
    fn input(&mut self, action: Action) {
        self.input = Buffer::new(action.text.clone());
        self.action = action;
        self.mode = Mode::Input;
    }
    fn matches(&self) -> Vec<usize> {
        let rows: Vec<(usize, String)> = match self.mode {
            Mode::Commands => COMMANDS
                .iter()
                .enumerate()
                .map(|(i, c)| (i, c.0.into()))
                .collect(),
            Mode::Search => self
                .visible()
                .into_iter()
                .map(|i| (i, self.editor.doc.tasks[i].text.clone()))
                .collect(),
            Mode::Recent => self
                .recent
                .iter()
                .enumerate()
                .map(|(i, f)| (i, f.path.display().to_string()))
                .collect(),
            _ => vec![],
        };
        let mut matches: Vec<_> = rows
            .iter()
            .filter_map(|(i, s)| {
                let score = if self.mode == Mode::Recent {
                    usize::from(s.to_lowercase().contains(&self.input.text.to_lowercase()))
                } else {
                    input::fuzzy(&self.input.text, s)
                };
                (score > 0).then_some((*i, score))
            })
            .collect();
        matches.sort_by_key(|(_, score)| std::cmp::Reverse(*score));
        matches.into_iter().map(|(i, _)| i).collect()
    }
    fn tags_available(&self) -> Vec<String> {
        self.editor
            .doc
            .tasks
            .iter()
            .flat_map(|t| t.tags.iter().cloned())
            .collect::<BTreeSet<_>>()
            .into_iter()
            .collect()
    }
    fn priorities_available(&self) -> Vec<i64> {
        self.editor
            .doc
            .tasks
            .iter()
            .map(|t| t.priority)
            .filter(|p| *p > 0)
            .collect::<BTreeSet<_>>()
            .into_iter()
            .collect()
    }
    fn open_versions(&mut self) -> Result<(), String> {
        self.versions = self.editor.store.versions()?;
        self.cursor = 0;
        self.scroll = 0;
        self.confirm = false;
        self.mode = Mode::Versions;
        self.preview_version()
    }
    fn preview_version(&mut self) -> Result<(), String> {
        self.preview = if let Some(v) = self.versions.get(self.cursor) {
            self.editor.store.version(v.id)?
        } else {
            String::new()
        };
        self.diff = diff_lines(&self.editor.doc.source, &self.preview);
        self.scroll = 0;
        Ok(())
    }
    fn command(&mut self, name: &str) {
        self.mode = Mode::Normal;
        self.cursor = 0;
        self.input = Buffer::default();
        match name {
            "check-all" | "uncheck-all" | "sort-done" | "sort-due" | "sort-priority"
            | "clear-done" => {
                self.apply(Action::new(name, self.selected, ""));
                if name == "clear-done" {
                    self.clear_sections();
                }
            }
            "filter-done" => {
                self.settings.filter_done = !self.settings.filter_done;
                self.clamp();
            }
            "filter-due" | "filter-overdue" | "filter-today" | "filter-week" => {
                let value = if name == "filter-due" {
                    "all"
                } else {
                    name.trim_start_matches("filter-")
                };
                self.due = if self.due == value {
                    String::new()
                } else {
                    value.into()
                };
                self.clamp();
            }
            "read-only" => {
                self.settings.read_only = !self.settings.read_only;
                self.editor.readonly = self.settings.read_only;
            }
            "save" => {
                let result = self.editor.manual_save();
                self.result(result);
            }
            "force-save" => {
                let result = self.editor.force_save();
                self.result(result);
            }
            "reload" => {
                let result = self.editor.reload();
                if result.is_ok() {
                    self.settings =
                        Settings::new(&self.config, &self.editor.doc.metadata, &self.flags);
                    self.editor.readonly = self.settings.read_only;
                    self.clear_sections();
                }
                self.result(result);
            }
            "versions" => {
                if let Err(e) = self.open_versions() {
                    self.status = e;
                }
            }
            "diff" => {
                if self.editor.dirty {
                    self.diff = diff_lines(
                        &std::fs::read_to_string(&self.path).unwrap_or_default(),
                        &self.editor.doc.source,
                    );
                    self.scroll = 0;
                    self.mode = Mode::Diff;
                } else {
                    self.status = "No unresolved file conflict".into();
                }
            }
            "wrap" => self.settings.word_wrap = !self.settings.word_wrap,
            "line-numbers" => self.line_numbers = !self.line_numbers,
            "show-headings" => self.settings.show_headings = !self.settings.show_headings,
            "set-max-visible" => self.mode = Mode::MaxVisible,
            "sections" => self.open_sections(),
            "all-sections" => self.clear_sections(),
            "theme" => {
                self.original_theme = self.theme.clone();
                self.cursor = self
                    .themes
                    .keys()
                    .position(|n| *n == self.theme)
                    .unwrap_or(0);
                self.mode = Mode::Theme;
            }
            _ => self.status = format!("Unknown command {name}"),
        }
    }
    fn open_sections(&mut self) {
        self.mode = Mode::Sections;
        self.cursor = self.section.unwrap_or_else(|| {
            self.editor
                .doc
                .headings
                .iter()
                .rposition(|h| h.before_todo_index <= self.selected)
                .unwrap_or(0)
        });
    }
    fn reload_if_idle(&mut self) -> bool {
        if self.mode != Mode::Normal || self.editor.dirty {
            return false;
        }
        match self.editor.store.changed() {
            Ok(true) => {
                self.command("reload");
                true
            }
            Err(e) => {
                self.status = e;
                true
            }
            _ => false,
        }
    }
    fn switch(&mut self, path: PathBuf) -> Result<(), String> {
        if self.editor.dirty && !self.settings.read_only {
            return Err(
                "Unsaved changes: :save, :reload or :force-save before switching files".into(),
            );
        }
        let dir = config::directory()?;
        let mut store = Store::load(&path)?;
        store.enable_history_with_limit(true, self.config.versioning.max_versions)?;
        let next = Editor::new(store, false)?;
        let _ = Recent::record(
            &dir,
            self.config.recent.max_files,
            &self.path,
            self.selected,
        );
        self.editor.store.finish()?;
        *self.editor = next;
        self.path = path;
        self.settings = Settings::new(&self.config, &self.editor.doc.metadata, &self.flags);
        self.editor.readonly = self.settings.read_only;
        self.tags.clear();
        self.priorities.clear();
        self.due.clear();
        self.section = None;
        self.folded.clear();
        self.selected = Recent::load(&dir, self.config.recent.max_files)?
            .cursor(&self.path)
            .unwrap_or(0);
        self.clamp();
        self.mode = Mode::Normal;
        self.status.clear();
        Ok(())
    }
    fn input_key(&mut self, key: KeyEvent) {
        if key
            .modifiers
            .intersects(KeyModifiers::CONTROL | KeyModifiers::SUPER)
            && matches!(key.code, KeyCode::Char('v' | 'y'))
        {
            match clipboard::paste() {
                Ok(s) => self.input.insert(&s),
                Err(e) => self.status = e,
            };
            return;
        }
        match key.code {
            KeyCode::Esc => {
                self.mode = if self.mode == Mode::Heading {
                    Mode::Sections
                } else {
                    Mode::Normal
                };
                self.input = Buffer::default();
                self.status = "Cancelled".into();
            }
            KeyCode::Enter => {
                if self.input.text.trim().is_empty() {
                    self.mode = Mode::Normal;
                    return;
                }
                if self.mode == Mode::MaxVisible {
                    match self.input.text.parse::<usize>() {
                        Ok(n) => {
                            self.settings.max_visible = n;
                            self.mode = Mode::Normal;
                        }
                        Err(_) => self.status = "Enter a non-negative number".into(),
                    };
                    return;
                }
                let mut action = self.action.clone();
                action.text = self.input.text.clone();
                let heading = self.mode == Mode::Heading;
                if heading && self.settings.read_only {
                    self.status = "read-only file: section editing disabled".into();
                    return;
                }
                self.mode = Mode::Normal;
                self.apply(action);
                if heading {
                    self.clear_sections();
                    self.mode = Mode::Sections;
                    self.cursor = self
                        .editor
                        .doc
                        .headings
                        .len()
                        .saturating_sub(1)
                        .min(self.cursor);
                }
                self.input = Buffer::default();
            }
            _ => {
                self.input.key(key);
            }
        }
    }
    fn picker_key(&mut self, key: KeyEvent) {
        let rows = self.matches();
        let down = key.code == KeyCode::Down
            || (key.modifiers.contains(KeyModifiers::CONTROL)
                && matches!(key.code, KeyCode::Char('n' | 'j')));
        let up = key.code == KeyCode::Up
            || (key.modifiers.contains(KeyModifiers::CONTROL)
                && matches!(key.code, KeyCode::Char('p' | 'k')));
        if down {
            self.cursor = if self.mode == Mode::Recent && !rows.is_empty() {
                (self.cursor + 1) % rows.len()
            } else {
                (self.cursor + 1).min(rows.len().saturating_sub(1))
            };
            return;
        }
        if up {
            self.cursor = if self.mode == Mode::Recent && !rows.is_empty() && self.cursor == 0 {
                rows.len() - 1
            } else {
                self.cursor.saturating_sub(1)
            };
            return;
        }
        match key.code {
            KeyCode::Esc => {
                self.mode = Mode::Normal;
                self.input = Buffer::default();
            }
            KeyCode::Enter => {
                if let Some(i) = rows.get(self.cursor).copied() {
                    match self.mode {
                        Mode::Commands => self.command(COMMANDS[i].0),
                        Mode::Search => {
                            self.selected = i;
                            self.mode = Mode::Normal;
                            self.input = Buffer::default();
                        }
                        Mode::Recent => {
                            let result = self.switch(self.recent[i].path.clone());
                            if let Err(e) = result {
                                self.status = e;
                            }
                        }
                        _ => {}
                    }
                }
            }
            KeyCode::Tab if self.mode == Mode::Commands => {
                if let Some(i) = rows.get(self.cursor) {
                    self.input = Buffer::new(COMMANDS[*i].0.into());
                    self.cursor = 0;
                }
            }
            _ => {
                if key.modifiers.contains(KeyModifiers::CONTROL)
                    && matches!(key.code, KeyCode::Char('y' | 'v'))
                {
                    if let Ok(s) = clipboard::paste() {
                        self.input.insert(&s);
                    }
                } else if self.input.key(key) {
                    self.cursor = 0;
                }
                if self.mode == Mode::Recent && self.input.text.starts_with(' ') {
                    self.input = Buffer::new(self.input.text.trim_start().into());
                }
            }
        }
    }
    fn overlay_key(&mut self, mut key: KeyEvent) {
        if key.modifiers.contains(KeyModifiers::CONTROL) {
            key.code = match key.code {
                KeyCode::Char('n' | 'j') => KeyCode::Down,
                KeyCode::Char('p' | 'k') => KeyCode::Up,
                KeyCode::Char('d') if self.mode == Mode::Versions => KeyCode::PageDown,
                KeyCode::Char('u') if self.mode == Mode::Versions => KeyCode::PageUp,
                _ => key.code,
            };
        }

        let len = match self.mode {
            Mode::Tags => self.tags_available().len(),
            Mode::Priorities => self.priorities_available().len(),
            Mode::Due => 4,
            Mode::Sections => self.editor.doc.headings.len(),
            Mode::Theme => self.themes.len(),
            Mode::Versions => self.versions.len(),
            _ => 0,
        };
        if matches!(key.code, KeyCode::Down | KeyCode::Char('j')) {
            self.cursor = (self.cursor + 1).min(len.saturating_sub(1));
            self.preview_overlay();
            return;
        }
        if matches!(key.code, KeyCode::Up | KeyCode::Char('k')) {
            self.cursor = self.cursor.saturating_sub(1);
            self.preview_overlay();
            return;
        }
        if matches!(key.code, KeyCode::Home | KeyCode::Char('g')) {
            self.cursor = 0;
            self.preview_overlay();
            return;
        }
        if matches!(key.code, KeyCode::End | KeyCode::Char('G')) {
            self.cursor = len.saturating_sub(1);
            self.preview_overlay();
            return;
        }
        match self.mode {
            Mode::Tags | Mode::Priorities | Mode::Due => match key.code {
                KeyCode::Esc => self.mode = Mode::Normal,
                KeyCode::Char('c') => {
                    match self.mode {
                        Mode::Tags => self.tags.clear(),
                        Mode::Priorities => self.priorities.clear(),
                        _ => self.due.clear(),
                    };
                    self.clamp();
                }
                KeyCode::Char(' ') | KeyCode::Enter => {
                    match self.mode {
                        Mode::Tags => {
                            if let Some(tag) = self.tags_available().get(self.cursor)
                                && !self.tags.remove(tag)
                            {
                                self.tags.insert(tag.clone());
                            }
                        }
                        Mode::Priorities => {
                            if let Some(p) = self.priorities_available().get(self.cursor)
                                && !self.priorities.remove(p)
                            {
                                self.priorities.insert(*p);
                            }
                        }
                        _ => {
                            let due = ["overdue", "today", "week", "all"][self.cursor];
                            self.due = if self.due == due {
                                String::new()
                            } else {
                                due.into()
                            };
                        }
                    }
                    self.clamp();
                    self.mode = Mode::Normal;
                }
                _ => {}
            },
            Mode::Sections => self.section_key(key),
            Mode::Theme => match key.code {
                KeyCode::Esc => {
                    self.theme = self.original_theme.clone();
                    self.mode = Mode::Normal;
                }
                KeyCode::Enter => {
                    let result = config::directory()
                        .and_then(|dir| config::save_theme_at(&dir, &self.theme));
                    self.result(result);
                    self.mode = Mode::Normal;
                }
                _ => {}
            },
            Mode::Versions => match key.code {
                KeyCode::Esc => {
                    self.mode = Mode::Normal;
                }
                KeyCode::Enter if len > 0 => {
                    self.confirm = true;
                }
                KeyCode::PageDown => self.scroll = self.scroll.saturating_add(10),
                KeyCode::PageUp => self.scroll = self.scroll.saturating_sub(10),
                _ => {}
            },
            _ => {}
        }
    }
    fn preview_overlay(&mut self) {
        if self.mode == Mode::Theme {
            if let Some(n) = self.themes.keys().nth(self.cursor) {
                self.theme = n.clone();
            }
        } else if self.mode == Mode::Versions
            && let Err(e) = self.preview_version()
        {
            self.status = e;
        }
    }
    fn section_key(&mut self, key: KeyEvent) {
        match key.code {
            KeyCode::Esc | KeyCode::Char('s') => self.mode = Mode::Normal,
            KeyCode::Char('a') => {
                self.clear_sections();
                self.mode = Mode::Normal;
            }
            KeyCode::Enter if !self.editor.doc.headings.is_empty() => {
                self.section = Some(self.cursor);
                self.folded.clear();
                self.settings.show_headings = true;
                self.clamp();
                self.mode = Mode::Normal;
            }
            KeyCode::Char(' ') if !self.editor.doc.headings.is_empty() => {
                if !self.folded.remove(&self.cursor) {
                    self.folded.insert(self.cursor);
                }
                self.clamp();
            }
            KeyCode::Char('e' | 'n' | 'N') => {
                if self.settings.read_only {
                    self.status = "read-only file: section editing disabled".into();
                    return;
                }
                let heading = self.editor.doc.headings.get(self.cursor);
                let (kind, index, level, text) = if key.code == KeyCode::Char('e') {
                    let Some(h) = heading else {
                        return;
                    };
                    (
                        "rename-heading",
                        self.cursor as isize,
                        h.level,
                        h.text.clone(),
                    )
                } else {
                    let level = heading
                        .map_or(1, |h| h.level + usize::from(key.code == KeyCode::Char('N')));
                    if level > 6 {
                        self.status = "Level 6 headings cannot have a subsection".into();
                        return;
                    }
                    (
                        "create-heading",
                        if heading.is_some() {
                            self.cursor as isize
                        } else {
                            -1
                        },
                        level,
                        String::new(),
                    )
                };
                self.action = Action {
                    kind: kind.into(),
                    index,
                    level,
                    text: text.clone(),
                    ..Action::default()
                };
                self.input = Buffer::new(text);
                self.mode = Mode::Heading;
            }
            _ => {}
        }
    }
    fn move_key(&mut self, key: KeyEvent) {
        match key.code {
            KeyCode::Esc => {
                if let Some((doc, index, dirty)) = self.moving.take() {
                    self.editor.doc = doc;
                    self.selected = index;
                    self.editor.dirty = dirty;
                }
                self.mode = Mode::Normal;
            }
            KeyCode::Enter => {
                if let Some((original, _, dirty)) = self.moving.take() {
                    let next = std::mem::replace(&mut self.editor.doc, original);
                    self.editor.dirty = dirty;
                    let result = self.editor.commit(next);
                    self.mode = Mode::Normal;
                    self.result(result);
                }
            }
            KeyCode::Up | KeyCode::Down | KeyCode::Char('j' | 'k') => {
                let down = matches!(key.code, KeyCode::Down | KeyCode::Char('j'));
                let visible = self.visible();
                let Some(position) = visible.iter().position(|i| *i == self.selected) else {
                    return;
                };
                let item = &self.editor.doc.items[self.editor.doc.tasks[self.selected].item];
                let target = if down {
                    visible.iter().skip(position + 1).copied().find(|i| {
                        self.editor.doc.items[self.editor.doc.tasks[*i].item]
                            .range
                            .start
                            >= item.range.end
                    })
                } else {
                    visible[..position].last().copied()
                };
                if let Some(target) = target {
                    let action = Action {
                        kind: "move-to-position".into(),
                        index: self.selected as isize,
                        target: target as isize,
                        insert_after: down,
                        ..Action::default()
                    };
                    let old = self.editor.doc.tasks[self.selected].clone();
                    match self.editor.doc.apply(&action) {
                        Ok((next, _)) => {
                            let count = self
                                .editor
                                .doc
                                .tasks
                                .iter()
                                .filter(|t| {
                                    self.editor.doc.items[t.item].range.start >= item.range.start
                                        && self.editor.doc.items[t.item].range.start
                                            < item.range.end
                                })
                                .count();
                            let destination = if down {
                                let target_end = self.editor.doc.items
                                    [self.editor.doc.tasks[target].item]
                                    .range
                                    .end;
                                self.editor
                                    .doc
                                    .tasks
                                    .iter()
                                    .take_while(|t| {
                                        self.editor.doc.items[t.item].range.start < target_end
                                    })
                                    .count()
                                    - count
                            } else {
                                target
                            };
                            self.editor.doc = next;
                            self.selected =
                                destination.min(self.editor.doc.tasks.len().saturating_sub(1));
                            debug_assert_eq!(self.editor.doc.tasks[self.selected].text, old.text);
                        }
                        Err(e) => self.status = e,
                    }
                }
            }
            _ => {}
        }
    }
    fn handle(&mut self, event: Event) -> bool {
        if let Event::Paste(s) = event {
            if matches!(
                self.mode,
                Mode::Input
                    | Mode::Search
                    | Mode::Commands
                    | Mode::Heading
                    | Mode::MaxVisible
                    | Mode::Recent
            ) {
                self.input.insert(&s);
                self.cursor = 0;
            }
            return false;
        }
        let Event::Key(key) = event else {
            return false;
        };
        if key.kind == KeyEventKind::Release {
            return false;
        }
        if key.modifiers.contains(KeyModifiers::CONTROL)
            && matches!(key.code, KeyCode::Char('c' | 'd'))
            && self.mode != Mode::Versions
        {
            if self.editor.dirty && !self.settings.read_only && !self.quit_confirm {
                self.status = "Unsaved changes. Quit again to discard, or :save / :reload.".into();
                self.quit_confirm = true;
                return false;
            }
            return true;
        }
        if self.mode == Mode::Versions && self.confirm {
            self.confirm = false;
            if matches!(key.code, KeyCode::Char('y' | 'Y')) {
                let id = self.versions[self.cursor].id;
                let result = self.editor.restore(id);
                if result.is_ok() {
                    self.settings =
                        Settings::new(&self.config, &self.editor.doc.metadata, &self.flags);
                    self.editor.readonly = self.settings.read_only;
                    self.clear_sections();
                    self.mode = Mode::Normal;
                }
                self.result(result);
            }
            return false;
        }
        match self.mode {
            Mode::Input | Mode::Heading | Mode::MaxVisible => {
                self.input_key(key);
                return false;
            }
            Mode::Search | Mode::Commands | Mode::Recent => {
                self.picker_key(key);
                return false;
            }
            Mode::Tags
            | Mode::Priorities
            | Mode::Due
            | Mode::Sections
            | Mode::Theme
            | Mode::Versions => {
                self.overlay_key(key);
                return false;
            }
            Mode::Move => {
                self.move_key(key);
                return false;
            }
            Mode::Help => {
                if key.code == KeyCode::Home {
                    self.scroll = 0;
                } else if key.code == KeyCode::End {
                    self.scroll = usize::MAX;
                } else if matches!(key.code, KeyCode::Esc | KeyCode::Char('?')) {
                    self.mode = Mode::Normal;
                } else if matches!(
                    key.code,
                    KeyCode::PageDown | KeyCode::Down | KeyCode::Char('j')
                ) {
                    self.scroll = self.scroll.saturating_add(5);
                } else if matches!(key.code, KeyCode::PageUp | KeyCode::Up | KeyCode::Char('k')) {
                    self.scroll = self.scroll.saturating_sub(5);
                }
                return false;
            }
            Mode::Diff => {
                match key.code {
                    KeyCode::Home => self.scroll = 0,
                    KeyCode::End => self.scroll = usize::MAX,
                    KeyCode::Esc => self.mode = Mode::Normal,
                    KeyCode::Down | KeyCode::PageDown | KeyCode::Char('j') => {
                        self.scroll = self.scroll.saturating_add(10)
                    }
                    KeyCode::Up | KeyCode::PageUp | KeyCode::Char('k') => {
                        self.scroll = self.scroll.saturating_sub(10)
                    }
                    _ => {}
                }
                return false;
            }
            Mode::Normal => {}
        }
        if let KeyCode::Char(c) = key.code
            && c.is_ascii_digit()
            && (c != '0' || !self.number.is_empty())
        {
            if self.number.len() < 8 {
                self.number.push(c);
            }
            return false;
        }
        let count = self.number.parse::<usize>().unwrap_or(1).max(1);
        self.number.clear();
        if key.code != KeyCode::Char('g') {
            self.g = false;
        }
        if !matches!(key.code, KeyCode::Esc | KeyCode::Char('q')) {
            self.quit_confirm = false;
        }
        let visible = self.visible();
        let position = visible.iter().position(|i| *i == self.selected);
        match key.code {
            KeyCode::Esc | KeyCode::Char('q') => {
                if self.editor.dirty && !self.settings.read_only && !self.quit_confirm {
                    self.status =
                        "Unsaved changes. Quit again to discard, or :save / :reload.".into();
                    self.quit_confirm = true;
                } else {
                    return true;
                }
            }
            KeyCode::Down | KeyCode::Char('j') => {
                if let Some(p) = position {
                    self.selected = visible[p.saturating_add(count).min(visible.len() - 1)];
                }
            }
            KeyCode::Up | KeyCode::Char('k') => {
                if let Some(p) = position {
                    self.selected = visible[p.saturating_sub(count)];
                }
            }
            KeyCode::PageDown => {
                if let Some(p) = position {
                    self.selected = visible[(p + 10).min(visible.len() - 1)];
                }
            }
            KeyCode::PageUp => {
                if let Some(p) = position {
                    self.selected = visible[p.saturating_sub(10)];
                }
            }
            KeyCode::Home => self.selected = visible.first().copied().unwrap_or(0),
            KeyCode::End | KeyCode::Char('G') => {
                self.selected = visible.last().copied().unwrap_or(0)
            }
            KeyCode::Char('g') => {
                if self.g {
                    self.selected = visible.first().copied().unwrap_or(0);
                    self.g = false;
                } else {
                    self.g = true;
                }
            }
            KeyCode::Enter | KeyCode::Char(' ') if position.is_some() => {
                let index = self.selected;
                self.apply(Action::new("toggle", index, ""));
                if !self.visible().contains(&index) {
                    self.selected = self.best_selection(index, false);
                }
            }
            KeyCode::Char('d') if position.is_some() => {
                let selection = self.best_selection(self.selected, true);
                self.apply(Action::new("delete", self.selected, ""));
                self.selected = selection.min(self.editor.doc.tasks.len().saturating_sub(1));
                self.clear_sections();
            }
            KeyCode::Char('e') if position.is_some() => self.input(Action::new(
                "edit",
                self.selected,
                &self.editor.doc.tasks[self.selected].text.clone(),
            )),
            KeyCode::Char('n' | 'N' | 'a') => {
                let kind = if key.code == KeyCode::Char('n') && position.is_some() {
                    "insert"
                } else if self.section.is_some() {
                    "add-in-section"
                } else {
                    "add"
                };
                let index = if kind == "add-in-section" {
                    self.section.unwrap()
                } else {
                    self.selected
                };
                self.input(Action::new(kind, index, ""));
            }
            KeyCode::Tab | KeyCode::BackTab if position.is_some() && !self.settings.read_only => {
                self.apply(Action::new(
                    if key.code == KeyCode::Tab {
                        "indent"
                    } else {
                        "outdent"
                    },
                    self.selected,
                    "",
                ))
            }
            KeyCode::Char('u') => {
                let result = self.editor.undo();
                self.clear_sections();
                self.result(result);
            }
            KeyCode::Char('m') if position.is_some() => {
                self.moving = Some((self.editor.doc.clone(), self.selected, self.editor.dirty));
                self.mode = Mode::Move;
            }
            KeyCode::Char('c') if position.is_some() => {
                self.status = match clipboard::copy(&self.editor.doc.tasks[self.selected].text) {
                    Ok(()) => "Copied to clipboard".into(),
                    Err(e) => e,
                };
            }
            KeyCode::Char('/') => {
                self.mode = Mode::Search;
                self.input = Buffer::default();
                self.cursor = 0;
            }
            KeyCode::Char(':') => {
                self.mode = Mode::Commands;
                self.input = Buffer::default();
                self.cursor = 0;
            }
            KeyCode::Char('t') => {
                self.mode = Mode::Tags;
                self.cursor = 0;
            }
            KeyCode::Char('p') => {
                self.mode = Mode::Priorities;
                self.cursor = 0;
            }
            KeyCode::Char('D') => {
                self.mode = Mode::Due;
                self.cursor = 0;
            }
            KeyCode::Char('s') => self.open_sections(),
            KeyCode::Char('S') => self.clear_sections(),
            KeyCode::Char('r') => {
                match config::directory()
                    .and_then(|dir| Recent::load(&dir, self.config.recent.max_files))
                {
                    Ok(r) => {
                        self.recent = r.files;
                        self.mode = Mode::Recent;
                        self.input = Buffer::default();
                        self.cursor = 0;
                    }
                    Err(e) => self.status = e,
                }
            }
            KeyCode::Char('v') => {
                if let Err(e) = self.open_versions() {
                    self.status = e;
                }
            }
            KeyCode::Char('?') => {
                self.mode = Mode::Help;
                self.scroll = 0;
            }
            _ => {}
        }
        false
    }
    fn mode_name(&self) -> &'static str {
        match self.mode {
            Mode::Normal => "TASKS",
            Mode::Move => "MOVE",
            Mode::Search => "SEARCH",
            Mode::Commands => "COMMAND",
            Mode::Recent => "RECENT",
            Mode::Theme => "THEME",
            Mode::Tags => "TAGS",
            Mode::Priorities => "PRIORITY",
            Mode::Due => "DUE DATE",
            Mode::Sections => "SECTIONS",
            Mode::Heading => "SECTION",
            Mode::Versions => "HISTORY",
            Mode::Diff => "CONFLICT",
            Mode::Help => "HELP",
            Mode::MaxVisible => "MAX VISIBLE",
            Mode::Input if self.action.kind == "edit" => "EDIT",
            Mode::Input => "NEW",
        }
    }
    fn context(&self) -> String {
        let mut parts = Vec::new();
        if self.settings.read_only {
            parts.push("READ ONLY".into());
        }
        if self.editor.dirty {
            parts.push("UNSAVED".into());
        }
        if let Some(h) = self.section.and_then(|i| self.editor.doc.headings.get(i)) {
            parts.push(format!("Section: {} · S all tasks", clean(&h.text)));
        }
        if !self.folded.is_empty() {
            parts.push(format!(
                "Folded: {}",
                self.folded
                    .iter()
                    .filter_map(|i| self.editor.doc.headings.get(*i))
                    .map(|h| clean(&h.text))
                    .collect::<Vec<_>>()
                    .join(", ")
            ));
        }
        if self.settings.filter_done {
            parts.push("open only".into());
        }
        parts.extend(self.tags.iter().map(|t| format!("#{t}")));
        parts.extend(self.priorities.iter().map(|p| format!("!p{p}")));
        if !self.due.is_empty() {
            parts.push(format!("due:{}", self.due));
        }
        if self.settings.word_wrap {
            parts.push("WRAP".into());
        }
        if self.settings.show_headings {
            parts.push("HEADINGS".into());
        }
        if self.settings.max_visible > 0 {
            parts.push(format!("MAX:{}", self.settings.max_visible));
        }
        parts.join(" · ")
    }
    fn panel(&self, title: String) -> Block<'static> {
        Block::bordered()
            .border_type(BorderType::Rounded)
            .title(title)
            .border_style(Style::default().fg(self.color("Dim")))
            .style(Style::default().fg(self.color("Base")))
    }
    fn draw(&mut self, frame: &mut Frame) {
        self.links.clear();
        let area = frame.area();
        frame.render_widget(
            Block::default().style(Style::default().fg(self.color("Base"))),
            area,
        );
        let areas = Layout::vertical([
            Constraint::Length(3),
            Constraint::Min(1),
            Constraint::Length(3),
        ])
        .split(area);
        let file = self
            .path
            .file_name()
            .unwrap_or(self.path.as_os_str())
            .to_string_lossy();
        let done = self.editor.doc.tasks.iter().filter(|t| t.checked).count();
        let visible = self.visible().len();
        let total = self.editor.doc.tasks.len();
        let heading = vec![
            Line::from(vec![
                Span::styled(
                    " tdx ",
                    Style::default()
                        .fg(self.color("Accent"))
                        .add_modifier(Modifier::BOLD),
                ),
                Span::raw(format!("{}  ", clean(&file))),
                Span::styled(
                    format!("v{}", crate::version()),
                    Style::default().fg(self.color("Dim")),
                ),
            ]),
            Line::from(vec![
                Span::styled(
                    format!(" {} open", total - done),
                    Style::default().fg(self.color("Accent")),
                ),
                Span::styled(
                    format!(" · {done} done"),
                    Style::default().fg(self.color("Success")),
                ),
                Span::raw(format!(" · {visible}/{total} visible")),
            ]),
            Line::styled(
                format!(" {}", self.context()),
                Style::default().fg(self.color(if self.editor.dirty { "Warning" } else { "Dim" })),
            ),
        ];
        frame.render_widget(Paragraph::new(heading), areas[0]);
        let input_mode = matches!(self.mode, Mode::Input | Mode::Heading | Mode::MaxVisible);
        let controls = if area.width < 80 {
            match self.mode {
                Mode::Normal => "n new · ␣ toggle · : cmd · ? help · Esc quit".into(),
                Mode::Commands | Mode::Search | Mode::Recent => {
                    "Type query · ↑↓ select · Enter · Esc".into()
                }
                Mode::Theme => "↑↓ preview · Enter apply · Esc cancel".into(),
                Mode::Sections => "Enter focus · n/N new · e rename · Esc".into(),
                _ => self.controls(),
            }
        } else {
            self.controls()
        };
        if input_mode {
            let window = presentation::input_window(
                &self.input.text,
                self.input.cursor,
                areas[2].width.saturating_sub(4) as usize,
            );
            frame.render_widget(
                Paragraph::new(window)
                    .block(
                        self.panel(format!(" {} · Enter save ", self.mode_name()))
                            .title_bottom(if areas[2].width >= 36 {
                                " Esc cancel · Ctrl-Y paste "
                            } else {
                                " Esc cancel "
                            }),
                    )
                    .style(Style::default().fg(self.color("Accent"))),
                areas[2],
            );
        } else {
            let status = if self.status.is_empty() {
                format!("{} · {}", self.mode_name(), self.path.display())
            } else {
                clean(&self.status)
            };
            frame.render_widget(
                Paragraph::new(vec![
                    Line::styled(controls, Style::default().fg(self.color("Dim"))),
                    Line::styled(
                        status,
                        Style::default().fg(self.color(if self.status.is_empty() {
                            "Accent"
                        } else {
                            "Warning"
                        })),
                    ),
                ])
                .wrap(Wrap { trim: false }),
                areas[2],
            );
        }
        match self.mode {
            Mode::Commands
            | Mode::Search
            | Mode::Recent
            | Mode::Tags
            | Mode::Priorities
            | Mode::Due
            | Mode::Theme => {
                self.draw_tasks(frame, areas[1]);
                let body = areas[1];
                let modal = if body.width >= 50 && body.height >= 10 {
                    let width = body.width.saturating_sub(6).min(82);
                    let count = match self.mode {
                        Mode::Commands | Mode::Search | Mode::Recent => self.matches().len(),
                        Mode::Tags => self.tags_available().len(),
                        Mode::Priorities => self.priorities_available().len(),
                        Mode::Due => 4,
                        _ => self.themes.len(),
                    };
                    let height = body
                        .height
                        .saturating_sub(2)
                        .min((count.max(1) + 2).min(13) as u16);
                    Rect::new(
                        body.x + (body.width - width) / 2,
                        body.y + body.height - height - 1,
                        width,
                        height,
                    )
                } else {
                    body
                };
                self.links
                    .retain(|&(x, y), _| !modal.contains((x, y).into()));
                frame.render_widget(Clear, modal);
                if matches!(self.mode, Mode::Commands | Mode::Search | Mode::Recent) {
                    self.draw_picker(frame, modal);
                } else {
                    self.draw_options(frame, modal);
                }
            }
            Mode::Sections | Mode::Heading => self.draw_sections(frame, areas[1]),
            Mode::Versions => {
                let panes = if areas[1].width < 76 {
                    Layout::vertical([
                        Constraint::Length(6.min(areas[1].height / 2)),
                        Constraint::Min(1),
                    ])
                    .split(areas[1])
                } else {
                    Layout::horizontal([Constraint::Length(29), Constraint::Min(1)]).split(areas[1])
                };
                let rows = self
                    .versions
                    .iter()
                    .map(|v| format!("#{} {}", v.id, v.created_at))
                    .collect();
                self.render_list(frame, panes[0], "FILE VERSION HISTORY", rows, self.cursor);
                self.draw_diff(frame, panes[1], "Current → saved version");
            }
            Mode::Diff => self.draw_diff(frame, areas[1], "Disk → local changes"),
            Mode::Help => {
                let width = areas[1].width.saturating_sub(2) as usize;
                let lines: Vec<Line<'static>> = help()
                    .lines()
                    .flat_map(|s| wrap(s, width, 0))
                    .map(Line::from)
                    .collect();
                let max = lines
                    .len()
                    .saturating_sub(areas[1].height.saturating_sub(2) as usize);
                self.scroll = self.scroll.min(max);
                frame.render_widget(
                    Paragraph::new(lines)
                        .block(self.panel(" Help · Home/End · PgUp/PgDn ".into()))
                        .scroll((self.scroll.min(u16::MAX as usize) as u16, 0)),
                    areas[1],
                );
            }
            _ => self.draw_tasks(frame, areas[1]),
        }
    }
    fn draw_diff(&mut self, frame: &mut Frame, area: Rect, title: &str) {
        let lines: Vec<_> = self
            .diff
            .iter()
            .flat_map(|line| {
                let input = line
                    .spans
                    .iter()
                    .flat_map(|span| presentation::glyphs(&span.content, span.style, None))
                    .collect();
                presentation::lines(input, area.width.saturating_sub(2) as usize, 0, true)
                    .iter()
                    .map(|g| presentation::line(g))
                    .collect::<Vec<_>>()
            })
            .collect();
        let max = lines
            .len()
            .saturating_sub(area.height.saturating_sub(2) as usize);
        self.scroll = self.scroll.min(max);
        frame.render_widget(
            Paragraph::new(lines)
                .block(self.panel(format!(" {title} · {}/{} ", self.scroll + 1, max + 1)))
                .scroll((self.scroll.min(u16::MAX as usize) as u16, 0)),
            area,
        );
    }
    fn controls(&self) -> String {
        match self.mode{
            Mode::Normal=>format!("j/k move · n/N new · e/d edit · u undo · / search · t/p/D filters · s sections · r recent · : commands · ? help{}{}{}{}",if self.tags.is_empty(){String::new()}else{format!(" #{}",self.tags.iter().cloned().collect::<Vec<_>>().join(" #"))},if self.priorities.is_empty(){String::new()}else{format!(" p{:?}",self.priorities)},if self.due.is_empty(){String::new()}else{format!(" due:{}",self.due)},if self.settings.filter_done{" · open only"}else{""}),
            Mode::Versions=>if self.confirm{"Restore this version? [y/N]"}else{"[↑/↓] Navigate  [PgUp/PgDn] Scroll  [Enter] Restore  [Esc] Close"}.into(),
            Mode::Diff=>"[PgUp/PgDn] Scroll · [Esc] Close · then :reload or :force-save".into(),
            Mode::Sections|Mode::Heading=>"j/k select · Enter focus · space fold · e rename · n section · N subsection · a all · Esc close".into(),
            Mode::Theme=>"↑/↓ live preview · Enter save theme · Esc restore previous theme".into(),
            Mode::Tags|Mode::Priorities|Mode::Due=>"j/k select · space/Enter toggle · c clear · Esc close".into(),
            Mode::Move=>"j/k move task · Enter commit · Esc cancel".into(),
            Mode::Search|Mode::Commands|Mode::Recent=>"Type to search · ↑/↓ or Ctrl-N/P select · Enter choose · Esc cancel".into(),
            _=>"PgUp/PgDn scroll · Esc close".into(),
        }
    }
    fn render_list(
        &self,
        frame: &mut Frame,
        area: Rect,
        title: &str,
        rows: Vec<String>,
        selected: usize,
    ) {
        let count = rows.len();
        let rows = if rows.is_empty() {
            vec![
                match self.mode {
                    Mode::Recent if self.input.text.is_empty() => "No recent files",
                    Mode::Recent => "No matching files",
                    Mode::Tags => "No tags found",
                    Mode::Priorities => "No priorities found",
                    Mode::Theme => "No themes available",
                    Mode::Sections => "No sections yet · n to create",
                    _ => "No matches",
                }
                .into(),
            ]
        } else {
            rows
        };
        let rows: Vec<_> = rows
            .into_iter()
            .map(|s| {
                let text = clean(&s);
                if matches!(self.mode, Mode::Search | Mode::Commands) {
                    ListItem::new(Line::from(highlight(
                        &text,
                        &self.input.text,
                        self.color("Accent"),
                    )))
                } else {
                    ListItem::new(text)
                }
            })
            .collect();
        let mut state = ListState::default();
        if count > 0 {
            state.select(Some(selected.min(rows.len() - 1)));
        }
        frame.render_stateful_widget(
            List::new(rows)
                .block(self.panel(format!(" {title} ")).title_bottom(format!(
                    " {}/{} ",
                    (selected + 1).min(count),
                    count
                )))
                .highlight_symbol(format!("{} ", self.config.display.select_marker))
                .highlight_style(
                    Style::default()
                        .fg(self.color("Accent"))
                        .add_modifier(Modifier::BOLD | Modifier::REVERSED),
                ),
            area,
            &mut state,
        );
    }
    fn draw_picker(&self, frame: &mut Frame, area: Rect) {
        let rows = self
            .matches()
            .into_iter()
            .map(|i| match self.mode {
                Mode::Commands => format!("{}  {}", COMMANDS[i].0, COMMANDS[i].1),
                Mode::Search => self.editor.doc.tasks[i].text.clone(),
                _ => {
                    let file = &self.recent[i];
                    let path = config::home()
                        .and_then(|home| {
                            file.path
                                .strip_prefix(home)
                                .ok()
                                .map(|p| format!("~/{}", p.display()))
                        })
                        .unwrap_or_else(|| file.path.display().to_string());
                    let info = format!(" ×{}", file.access_count);
                    let width = (area.width as usize).saturating_sub(
                        3 + self.config.display.select_marker.width() + info.width(),
                    );
                    format!(
                        "{}{}",
                        presentation::tail(&clean(&path), width.min(60)),
                        info
                    )
                }
            })
            .collect();
        let title = format!(
            "{}: {}",
            self.mode_name(),
            presentation::input_window(
                &self.input.text,
                self.input.cursor,
                area.width.saturating_sub(self.mode_name().len() as u16 + 7) as usize
            )
        );
        self.render_list(frame, area, &title, rows, self.cursor);
    }
    fn draw_options(&self, frame: &mut Frame, area: Rect) {
        let (title, rows) = match self.mode {
            Mode::Tags => (
                "Tags",
                self.tags_available()
                    .into_iter()
                    .map(|t| {
                        format!(
                            "[{}] #{}",
                            if self.tags.contains(&t) { "✓" } else { " " },
                            t
                        )
                    })
                    .collect(),
            ),
            Mode::Priorities => (
                "Priorities",
                self.priorities_available()
                    .into_iter()
                    .map(|p| {
                        format!(
                            "[{}] !p{}",
                            if self.priorities.contains(&p) {
                                "✓"
                            } else {
                                " "
                            },
                            p
                        )
                    })
                    .collect(),
            ),
            Mode::Due => (
                "Due dates",
                [
                    ("overdue", "Overdue", "Past due date"),
                    ("today", "Today", "Due today"),
                    ("week", "This Week", "Due within 7 days"),
                    ("all", "Has Due Date", "Any due date set"),
                ]
                .iter()
                .map(|(value, label, description)| {
                    format!(
                        "[{}] {}  {}",
                        if self.due == *value { "✓" } else { " " },
                        label,
                        description
                    )
                })
                .collect(),
            ),
            _ => (
                "Themes",
                self.themes
                    .keys()
                    .map(|name| {
                        format!("[{}] {}", if *name == self.theme { "●" } else { " " }, name)
                    })
                    .collect(),
            ),
        };
        self.render_list(frame, area, title, rows, self.cursor);
    }
    fn draw_sections(&self, frame: &mut Frame, area: Rect) {
        let rows = self
            .editor
            .doc
            .headings
            .iter()
            .enumerate()
            .map(|(i, h)| {
                let range = self.editor.doc.section_bounds(i);
                let total = range.len();
                let done = self.editor.doc.tasks[range]
                    .iter()
                    .filter(|t| t.checked)
                    .count();
                format!(
                    "{}{} {}  {}/{} done",
                    "  ".repeat(h.level - 1),
                    if self.folded.contains(&i) {
                        "▸"
                    } else {
                        "▾"
                    },
                    h.text,
                    done,
                    total
                )
            })
            .collect();
        self.render_list(
            frame,
            area,
            "Sections · n creates an empty section",
            rows,
            self.cursor,
        );
    }
    fn draw_tasks(&mut self, frame: &mut Frame, area: Rect) {
        let mut visible = self.visible();
        let matched = visible.len();
        let selected_pos = visible
            .iter()
            .position(|i| *i == self.selected)
            .unwrap_or(0);
        if self.settings.max_visible > 0 && visible.len() > self.settings.max_visible {
            let start = selected_pos
                .saturating_sub(self.settings.max_visible / 2)
                .min(visible.len() - self.settings.max_visible);
            visible = visible[start..start + self.settings.max_visible].to_vec();
        }
        // At most one screen of candidate tasks can be visible. Retain subtree
        // indexes, but avoid shaping and allocating every glyph in large files.
        let limit = (area.height as usize).max(1);
        if visible.len() > limit {
            let position = visible
                .iter()
                .position(|i| *i == self.selected)
                .unwrap_or(0);
            let start = position
                .saturating_sub(limit / 2)
                .min(visible.len() - limit);
            visible = visible[start..start + limit].to_vec();
        }
        let mut rows = Vec::new();
        let mut rich_rows: Vec<Vec<Vec<presentation::Glyph>>> = Vec::new();
        let mut selected_row = None;
        let mut last_heading = None;
        for i in visible {
            if self.settings.show_headings {
                let current = self
                    .editor
                    .doc
                    .headings
                    .iter()
                    .enumerate()
                    .rfind(|(_, h)| h.before_todo_index <= i)
                    .map(|(h, _)| h);
                if current != last_heading {
                    if let Some(h) = current {
                        let h = &self.editor.doc.headings[h];
                        rich_rows.push(vec![vec![]]);
                        rows.push(ListItem::new(Line::styled(
                            format!("{} {}", "#".repeat(h.level), clean(&h.text)),
                            Style::default().fg(self.color("Accent")),
                        )));
                    }
                    last_heading = current;
                }
            }
            if i == self.selected {
                selected_row = Some(rows.len());
            }
            let task = &self.editor.doc.tasks[i];
            let prefix = format!(
                "{}{}[{}] ",
                if self.line_numbers {
                    format!(
                        "{:>3} ",
                        if i == self.selected {
                            i + 1
                        } else {
                            i.abs_diff(self.selected)
                        }
                    )
                } else {
                    String::new()
                },
                "  ".repeat(task.depth),
                if task.checked {
                    &self.config.display.check_symbol
                } else {
                    " "
                }
            );
            let style =
                Style::default().fg(self.color(if task.checked { "Important" } else { "Base" }));
            let marker_width = self.config.display.select_marker.width() + 1;
            let available = (area.width as usize).saturating_sub(2 + marker_width);
            let mut rich = presentation::glyphs(&prefix, style, None);
            rich.extend(presentation::inline(
                &clean(&task.text),
                style,
                style.fg(self.color("Accent")),
                style.bg(self.color("Dim")),
                |s| styled_text(s, style, self),
            ));
            let lines =
                presentation::lines(rich, available, prefix.width(), self.settings.word_wrap);
            rows.push(ListItem::new(
                lines
                    .iter()
                    .map(|g| presentation::line(g))
                    .collect::<Vec<_>>(),
            ));
            rich_rows.push(lines);
        }
        if rows.is_empty() {
            rows.push(ListItem::new(
                "No matching tasks · n creates a task · s browses empty sections",
            ));
        }
        let mut state = ListState::default();
        state.select(selected_row);
        frame.render_stateful_widget(
            List::new(rows)
                .block(self.panel(format!(
                    " Tasks · {}/{} ",
                    selected_pos + usize::from(matched > 0),
                    matched
                )))
                .highlight_symbol(format!("{} ", self.config.display.select_marker))
                .highlight_style(
                    Style::default()
                        .fg(self.color("Accent"))
                        .add_modifier(Modifier::BOLD | Modifier::REVERSED),
                ),
            area,
            &mut state,
        );
        let mut y = area.y.saturating_add(1);
        let start_x = area
            .x
            .saturating_add(1 + self.config.display.select_marker.width() as u16 + 1);
        for lines in rich_rows.iter().skip(state.offset()) {
            for line in lines {
                if y >= area.bottom().saturating_sub(1) {
                    break;
                }
                let mut x = start_x;
                for g in line {
                    let width = g.text.width() as u16;
                    if x.saturating_add(width) > area.right().saturating_sub(1) {
                        break;
                    }
                    if let Some(url) = &g.url {
                        self.links.insert((x, y), url.clone());
                    }
                    x = x.saturating_add(width);
                }
                y = y.saturating_add(1);
            }
        }
    }
}
fn clean(s: &str) -> String {
    s.chars()
        .map(|c| if c.is_control() { '�' } else { c })
        .collect()
}
fn wrap(s: &str, width: usize, indent: usize) -> Vec<String> {
    if width < 4 {
        return vec![s.into()];
    }
    let mut lines = Vec::new();
    let mut remaining = s.to_owned();
    let padding = " ".repeat(indent.min(width / 2));
    loop {
        let mut columns = 0;
        let mut boundary = remaining.len();
        let mut space = None;
        for (i, c) in remaining.char_indices() {
            if columns + c.width().unwrap_or(0) > width {
                boundary = i;
                break;
            }
            columns += c.width().unwrap_or(0);
            if c.is_whitespace() && i > padding.len() {
                space = Some(i);
            }
        }
        if boundary == remaining.len() {
            lines.push(remaining);
            break;
        }
        let split = space.unwrap_or(boundary);
        lines.push(remaining[..split].trim_end().into());
        remaining = format!("{padding}{}", remaining[split..].trim_start());
    }
    lines
}

fn styled_text(s: &str, base: Style, app: &App<'_>) -> Vec<Span<'static>> {
    static RE: std::sync::OnceLock<regex::Regex> = std::sync::OnceLock::new();
    let re = RE.get_or_init(|| {
        regex::Regex::new(r"#[a-zA-Z0-9_-]+|!p[0-9]+|@due\([0-9]{4}-[0-9]{2}-[0-9]{2}\)").unwrap()
    });
    let mut spans = Vec::new();
    let mut end = 0;
    for found in re.find_iter(s) {
        spans.push(Span::styled(s[end..found.start()].to_owned(), base));
        let part = found.as_str();
        let color = if part.starts_with('#') {
            "Tag"
        } else if let Some(priority) = part.strip_prefix("!p") {
            match priority.parse::<u64>().unwrap_or(0) {
                1 => "PriorityHigh",
                2 => "PriorityMedium",
                _ => "PriorityLow",
            }
        } else {
            let d = &part[5..part.len() - 1];
            let Ok(date) = NaiveDate::parse_from_str(d, "%Y-%m-%d") else {
                spans.push(Span::styled(part.to_owned(), base));
                end = found.end();
                continue;
            };
            // Match Go's time.Now().Truncate(24*time.Hour) date boundary.
            let days = (date - chrono::Utc::now().date_naive()).num_days();
            if days <= 0 {
                "DueUrgent"
            } else if days <= 3 {
                "DueSoon"
            } else {
                "DueFuture"
            }
        };
        spans.push(Span::styled(part.to_owned(), base.fg(app.color(color))));
        end = found.end();
    }
    spans.push(Span::styled(s[end..].to_owned(), base));
    spans
}

fn diff_lines(old: &str, new: &str) -> Vec<Line<'static>> {
    let diff = TextDiff::configure()
        .timeout(Duration::from_millis(500))
        .diff_lines(old, new);
    let mut lines = Vec::new();
    for op in diff.ops() {
        for change in diff.iter_inline_changes(op) {
            let (prefix, color) = match change.tag() {
                ChangeTag::Delete => ("- ", Color::Red),
                ChangeTag::Insert => ("+ ", Color::Green),
                _ => ("  ", Color::DarkGray),
            };
            let mut spans = vec![Span::styled(prefix, Style::default().fg(color))];
            for (emphasized, text) in change.iter_strings_lossy() {
                let style = Style::default().fg(color);
                spans.push(Span::styled(
                    clean(text.trim_end_matches(['\r', '\n'])),
                    if emphasized {
                        style.add_modifier(Modifier::BOLD | Modifier::UNDERLINED)
                    } else {
                        style
                    },
                ));
            }
            lines.push(Line::from(spans));
        }
    }
    if old == new {
        lines.insert(0, Line::from("No differences"));
    }
    lines
}
fn help() -> String {
    format!(
        "{}\n\nCommands\n{}\n\nMove mode groups all moves into one undo entry. Read-only checklist edits stay in memory; :save writes explicitly. Sections support focus/folding, rename and sibling/subsection creation. CLI flags override file frontmatter, then global settings. Version and conflict diffs highlight changed characters.\n",
        crate::HELP,
        COMMANDS
            .iter()
            .map(|(n, d)| format!("{n}: {d}"))
            .collect::<Vec<_>>()
            .join("\n")
    )
}
struct PasteGuard;
impl Drop for PasteGuard {
    fn drop(&mut self) {
        let _ = execute!(io::stdout(), DisableBracketedPaste);
    }
}
pub fn run(editor: &mut Editor, path: &Path, config: Config, flags: Overrides) -> io::Result<()> {
    ratatui::run(|terminal| {
        execute!(io::stdout(), EnableBracketedPaste)?;
        let _guard = PasteGuard;
        let mut app = App::new(editor, path, config, flags);
        let mut checked = Instant::now();
        #[cfg(windows)]
        let mut input = crate::console_input::Reader::default();
        let mut redraw = true;
        let mut previous_links = Links::new();
        loop {
            if redraw {
                let completed = terminal.draw(|f| app.draw(f))?;
                presentation::sync_links(
                    completed.buffer,
                    &app.links,
                    &previous_links,
                    &mut io::stdout(),
                )?;
                previous_links.clone_from(&app.links);
            }
            redraw = false;
            #[cfg(windows)]
            let event = input.next(Duration::from_millis(100))?;
            #[cfg(not(windows))]
            let event = if crossterm::event::poll(Duration::from_millis(100))? {
                Some(crossterm::event::read()?)
            } else {
                None
            };
            if let Some(event) = event {
                if app.handle(event) {
                    break;
                }
                redraw = true;
            }
            if checked.elapsed() >= Duration::from_millis(500) {
                redraw |= app.reload_if_idle();
                checked = Instant::now();
            }
        }
        if let Ok(dir) = config::directory()
            && app.path.exists()
        {
            Recent::record(&dir, app.config.recent.max_files, &app.path, app.selected)
                .map_err(io::Error::other)?;
        }
        Ok(())
    })
}

// Script input uses the identical state machine, with no raw-mode side effects.
pub fn piped(editor: &mut Editor, path: &Path, config: Config, flags: Overrides) -> io::Result<()> {
    use io::Read;
    let mut input = String::new();
    io::stdin().read_to_string(&mut input)?;
    let mut app = App::new(editor, path, config, flags);
    for event in decode_input(&input) {
        if app.handle(event) {
            break;
        }
    }
    let backend = ratatui::backend::TestBackend::new(100, 40);
    let mut terminal = ratatui::Terminal::new(backend).unwrap();
    terminal.draw(|f| app.draw(f)).unwrap();
    let buffer = terminal.backend().buffer();
    for y in 0..40 {
        let mut line = String::new();
        for x in 0..100 {
            line.push_str(buffer[(x, y)].symbol());
        }
        println!("{}", line.trim_end());
    }
    if app.path.exists() {
        Recent::record(
            &config::directory().map_err(io::Error::other)?,
            app.config.recent.max_files,
            &app.path,
            app.selected,
        )
        .map_err(io::Error::other)?;
    }
    Ok(())
}
fn decode_input(mut text: &str) -> Vec<Event> {
    let mut events = Vec::new();
    while !text.is_empty() {
        if let Some(rest) = text.strip_prefix("\x1b[200~") {
            if let Some((paste, next)) = rest.split_once("\x1b[201~") {
                events.push(Event::Paste(paste.into()));
                text = next;
                continue;
            }
            break;
        }
        let mut sequence = false;
        for (prefix, code) in [
            ("\x1b[A", KeyCode::Up),
            ("\x1b[B", KeyCode::Down),
            ("\x1b[C", KeyCode::Right),
            ("\x1b[D", KeyCode::Left),
            ("\x1b[H", KeyCode::Home),
            ("\x1b[F", KeyCode::End),
            ("\x1b[3~", KeyCode::Delete),
            ("\x1b[5~", KeyCode::PageUp),
            ("\x1b[6~", KeyCode::PageDown),
            ("\x1b[Z", KeyCode::BackTab),
        ] {
            if let Some(rest) = text.strip_prefix(prefix) {
                events.push(Event::Key(KeyEvent::new(code, KeyModifiers::NONE)));
                text = rest;
                sequence = true;
                break;
            }
        }
        if sequence {
            continue;
        }
        let ch = text.chars().next().unwrap();
        text = &text[ch.len_utf8()..];
        let (code, modifiers) = match ch {
            '\r' | '\n' => (KeyCode::Enter, KeyModifiers::NONE),
            '\t' => (KeyCode::Tab, KeyModifiers::NONE),
            '\x1b' => (KeyCode::Esc, KeyModifiers::NONE),
            '\x7f' => (KeyCode::Backspace, KeyModifiers::NONE),
            '\x01'..='\x1a' => (
                KeyCode::Char((ch as u8 + b'a' - 1) as char),
                KeyModifiers::CONTROL,
            ),
            c => (KeyCode::Char(c), KeyModifiers::NONE),
        };
        events.push(Event::Key(KeyEvent::new(code, modifiers)));
    }
    events
}

fn highlight(text: &str, query: &str, color: Color) -> Vec<Span<'static>> {
    let folded = query.to_lowercase();
    let mut wanted = folded.chars().peekable();
    text.chars()
        .map(|c| {
            let matched = wanted
                .peek()
                .is_some_and(|q| c.to_lowercase().any(|v| v == *q));
            if matched {
                wanted.next();
            }
            Span::styled(
                c.to_string(),
                if matched {
                    Style::default()
                        .fg(color)
                        .add_modifier(Modifier::BOLD | Modifier::UNDERLINED)
                } else {
                    Style::default()
                },
            )
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    const SOURCE: &str = "# Work\n\n- [ ] Alpha #work !p2\n  - [ ] Child #home !p1\n- [x] Beta #home !p3\n- [ ] Gamma #work !p1\n\n## Empty\n";
    fn with_app(f: impl FnOnce(&mut App<'_>)) {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("tasks.md");
        std::fs::write(&path, SOURCE).unwrap();
        let store = Store::with_lock_root(&path, dir.path().join("locks")).unwrap();
        let mut editor = Editor::new(store, false).unwrap();
        let mut app = App::new(&mut editor, &path, Config::default(), Overrides::default());
        f(&mut app);
    }
    fn keys(app: &mut App<'_>, input: &str) {
        for event in decode_input(input) {
            assert!(!app.handle(event), "unexpected quit: {input}");
        }
    }
    #[test]
    fn filtered_actions_use_full_document_indexes() {
        with_app(|app| {
            keys(app, "t ");
            assert_eq!(app.mode, Mode::Normal);
            assert_eq!(app.visible(), vec![1, 2]);
            keys(app, " ");
            assert!(app.editor.doc.tasks[1].checked);
            assert!(!app.editor.doc.tasks[0].checked);
            keys(app, "u");
            assert!(!app.editor.doc.tasks[1].checked);
        });
    }
    #[test]
    fn move_cancel_commit_and_undo_preserve_subtree() {
        with_app(|app| {
            keys(app, "mj\x1b");
            assert_eq!(app.editor.doc.source, SOURCE);
            keys(app, "mj\r");
            assert_eq!(app.editor.doc.tasks[0].text, "Beta #home !p3");
            assert_eq!(app.editor.doc.tasks[2].parent_index, Some(2));
            keys(app, "u");
            assert_eq!(app.editor.doc.source, SOURCE);
        });
    }
    #[test]
    fn readonly_edits_manual_save_and_metadata_flags() {
        with_app(|app| {
            app.command("read-only");
            keys(app, " ");
            assert!(app.editor.dirty);
            assert_eq!(std::fs::read_to_string(&app.path).unwrap(), SOURCE);
            app.command("save");
            assert!(!app.editor.dirty);
            assert!(
                std::fs::read_to_string(&app.path)
                    .unwrap()
                    .contains("[x] Alpha")
            );
            keys(app, "u");
            assert_eq!(app.editor.doc.source, SOURCE);
        });
    }
    #[test]
    fn sections_can_focus_empty_create_and_cancel() {
        with_app(|app| {
            keys(app, "sG\r");
            assert!(app.visible().is_empty());
            keys(app, "NAdded\r");
            assert_eq!(app.editor.doc.tasks.last().unwrap().text, "Added");
            keys(app, "Sse\x01Renamed \r");
            assert_eq!(app.mode, Mode::Sections);
            assert!(
                app.editor
                    .doc
                    .headings
                    .iter()
                    .any(|h| h.text.starts_with("Renamed"))
            );
        });
    }
    fn screen(app: &mut App<'_>, width: u16, height: u16) -> ratatui::buffer::Buffer {
        let mut terminal =
            ratatui::Terminal::new(ratatui::backend::TestBackend::new(width, height)).unwrap();
        terminal.draw(|f| app.draw(f)).unwrap();
        terminal.backend().buffer().clone()
    }
    fn text(buffer: &ratatui::buffer::Buffer) -> String {
        (0..buffer.area.height)
            .map(|y| {
                (0..buffer.area.width)
                    .map(|x| buffer[(x, y)].symbol())
                    .collect::<String>()
            })
            .collect::<Vec<_>>()
            .join("\n")
    }
    #[test]
    fn recent_picker_preserves_substring_order_wraps_and_shows_file_info() {
        with_app(|app| {
            app.mode = Mode::Recent;
            app.recent = ["/deep/aXb.md", "/deep/ab.md", "/deep/AB-final.md"]
                .into_iter()
                .map(|path| RecentFile {
                    path: PathBuf::from(path),
                    last_accessed: chrono::Utc::now(),
                    access_count: 7,
                    last_cursor_pos: 0,
                    content_hash: String::new(),
                    last_modified: chrono::Utc::now(),
                })
                .collect();
            app.input = Buffer::new("ab".into());
            assert_eq!(app.matches(), vec![1, 2]);
            app.picker_key(KeyEvent::new(KeyCode::Up, KeyModifiers::NONE));
            assert_eq!(app.cursor, 1);
            app.picker_key(KeyEvent::new(KeyCode::Down, KeyModifiers::NONE));
            assert_eq!(app.cursor, 0);
            app.recent[1].path = PathBuf::from(format!("/{}/ab-café🦀.md", "long/".repeat(30)));
            let shown = text(&screen(app, 48, 24));
            assert!(shown.contains("ab-café🦀") && shown.contains("×7") && shown.contains("1/2"));
            app.input = Buffer::new("missing".into());
            let shown = text(&screen(app, 48, 24));
            assert!(shown.contains("No matching files") && shown.contains("0/0"));
        });
        assert_eq!(presentation::tail("deep/café.md", 8), "…café.md");
        assert_eq!(
            presentation::tail("deep/cafe\u{301}.md", 8),
            "…cafe\u{301}.md"
        );
        assert_eq!(presentation::tail("deep/🦀.md", 6), "…🦀.md");
        assert_eq!(presentation::tail("deep/file", 0), "");
    }
    #[test]
    fn option_pickers_expose_active_theme_and_due_meanings() {
        with_app(|app| {
            app.mode = Mode::Due;
            app.due = "week".into();
            let shown = text(&screen(app, 100, 28));
            assert!(shown.contains("[✓] This Week") && shown.contains("Due within 7 days"));
            app.mode = Mode::Theme;
            app.cursor = app
                .themes
                .keys()
                .position(|name| *name == app.theme)
                .unwrap();
            let shown = text(&screen(app, 100, 28));
            assert!(shown.contains(&format!("[●] {}", app.theme)));
        });
    }
    #[test]
    fn metadata_rendering_matches_go_token_boundaries_and_invalid_dates() {
        with_app(|app| {
            let base = Style::default().fg(Color::White);
            let spans = styled_text("(#tag) !p01 !p12 @due(2026-99-99)", base, app);
            for (token, color) in [
                ("#tag", app.color("Tag")),
                ("!p01", app.color("PriorityHigh")),
                ("!p12", app.color("PriorityLow")),
                ("@due(2026-99-99)", Color::White),
            ] {
                assert_eq!(
                    spans.iter().find(|s| s.content == token).unwrap().style.fg,
                    Some(color)
                );
            }
        });
    }
    #[test]
    fn command_registry_matches_every_go_command() {
        let source = include_str!("../../../internal/tui/commands.go");
        let re = regex::Regex::new(r#"Name:\s*"([^"]+)""#).unwrap();
        let go: BTreeSet<_> = re.captures_iter(source).map(|c| c[1].to_owned()).collect();
        let rust: BTreeSet<_> = COMMANDS.iter().map(|c| c.0.to_owned()).collect();
        assert_eq!(go, rust);
    }
    #[test]
    fn presentation_exposes_context_cursor_links_and_full_scroll_content() {
        with_app(|app| {
            app.tags.insert("work".into());
            app.section = Some(0);
            let shown = text(&screen(app, 100, 28));
            assert!(
                shown.contains("3 open")
                    && shown.contains("1 done")
                    && shown.contains("Section: Work")
                    && shown.contains("#work")
            );
            app.tags.clear();
            app.section = None;
            app.editor.doc =
                Document::parse("- [ ] [Ratatui guide](https://ratatui.rs)\n".into()).unwrap();
            let shown = screen(app, 48, 24);
            assert!(text(&shown).contains("Ratatui guide") && !text(&shown).contains("](https"));
            let ((x, y), url) = app.links.first_key_value().unwrap();
            assert_eq!(url, "https://ratatui.rs");
            assert_eq!(shown[(*x, *y)].symbol(), "R");
            app.mode = Mode::Commands;
            screen(app, 80, 28);
            app.mode = Mode::Input;
            app.action.kind = "edit".into();
            app.input = Buffer::new(format!("{} café 🦀 final", "long ".repeat(30)));
            let input = text(&screen(app, 48, 24));
            assert!(input.contains("final▏") && input.contains("Esc cancel"));
            app.mode = Mode::Help;
            app.scroll = usize::MAX;
            let shown = text(&screen(app, 24, 20));
            assert!(app.scroll > 50 && shown.contains("characters."));
            app.mode = Mode::Diff;
            app.diff = diff_lines("old", &"long changed line ".repeat(60));
            app.scroll = usize::MAX;
            screen(app, 24, 20);
            assert!(app.scroll > app.diff.len());
            let end = app.scroll;
            screen(app, 24, 20);
            assert_eq!(app.scroll, end);
        });
    }
    #[test]
    #[ignore = "exports actual Ratatui buffers for visual review"]
    fn export_ratatui_gallery() {
        with_app(|app| {
            app.editor.doc = Document::parse("# Product launch\n\n- [x] Agree on the launch scope #planning\n- [ ] Review the [Ratatui guide](https://ratatui.rs) #design !p1\n  - [ ] Check keyboard navigation and focus !p2\n  - [ ] Test small terminals and Unicode café\n- [ ] Ship the release notes #writing @due(2026-09-09)\n\n## Engineering\n\n- [ ] Verify restore and concurrent saves #quality\n- [ ] Run `cargo test` on all platforms #quality\n\n## Later\n\n- [ ] Explore render performance #research\n".into()).unwrap();
            app.path = PathBuf::from("launch.md");
            app.settings.show_headings = true;
            app.selected = 1;
            let mut screens = Vec::new();
            for (name, mode, width, height) in [
                ("tasks", Mode::Normal, 100, 28),
                ("commands", Mode::Commands, 100, 28),
                ("sections", Mode::Sections, 100, 28),
                ("history", Mode::Versions, 100, 28),
                ("narrow", Mode::Normal, 48, 24),
                ("input", Mode::Input, 48, 24),
            ] {
                app.mode = mode;
                app.input = if mode == Mode::Commands {
                    Buffer::new("sort".into())
                } else {
                    Buffer::new("Review the release announcement and final screenshots".into())
                };
                app.action.kind = "edit".into();
                app.cursor = 0;
                app.versions = vec![
                    Version {
                        id: 3,
                        created_at: "2026-09-05 16:45".into(),
                    },
                    Version {
                        id: 2,
                        created_at: "2026-09-05 16:32".into(),
                    },
                    Version {
                        id: 1,
                        created_at: "2026-09-05 16:20".into(),
                    },
                ];
                app.diff = diff_lines(
                    "- [ ] Draft launch notes\n- [ ] Review screenshots\n",
                    "- [x] Draft launch notes\n- [ ] Review the final screenshots\n- [ ] Publish the announcement\n",
                );
                let buffer = screen(app, width, height);
                let cells: Vec<_> = buffer.content().iter().map(|c| serde_json::json!({"text":c.symbol(),"fg":format!("{:?}",c.fg),"bg":format!("{:?}",c.bg),"modifiers":format!("{:?}",c.modifier)})).collect();
                screens.push(
                    serde_json::json!({"name":name,"width":width,"height":height,"cells":cells}),
                );
            }
            let directory = std::env::var("TDX_GALLERY_DIR").expect("set TDX_GALLERY_DIR");
            std::fs::create_dir_all(&directory).unwrap();
            std::fs::write(
                Path::new(&directory).join("ratatui-screens.json"),
                serde_json::to_string_pretty(&screens).unwrap(),
            )
            .unwrap();
        });
    }
    #[test]
    fn overlays_render_at_narrow_and_wide_sizes() {
        with_app(|app| {
            for mode in [
                Mode::Normal,
                Mode::Input,
                Mode::Search,
                Mode::Commands,
                Mode::Tags,
                Mode::Priorities,
                Mode::Due,
                Mode::Sections,
                Mode::Heading,
                Mode::Recent,
                Mode::Theme,
                Mode::Versions,
                Mode::Diff,
                Mode::Help,
                Mode::MaxVisible,
                Mode::Move,
            ] {
                app.mode = mode;
                for width in [12, 24, 80, 160] {
                    let mut terminal =
                        ratatui::Terminal::new(ratatui::backend::TestBackend::new(width, 20))
                            .unwrap();
                    terminal.draw(|f| app.draw(f)).unwrap();
                    assert_eq!(terminal.backend().buffer().area.width, width);
                }
            }
        });
    }
    #[test]
    fn idle_reload_defers_local_input_and_applies_file_settings() {
        with_app(|app| {
            keys(app, "e");
            let external =
                "---\nread-only: true\nshow-headings: true\n---\n# Changed\n\n- [ ] External\n";
            std::fs::write(&app.path, external).unwrap();
            assert!(!app.reload_if_idle());
            assert_eq!(app.editor.doc.source, SOURCE);
            keys(app, "\x1b");
            assert!(app.reload_if_idle());
            assert!(app.settings.read_only && app.settings.show_headings);
            assert_eq!(app.editor.doc.source, external);
        });
    }
    #[test]
    fn utf8_words_highlights_and_inline_diff() {
        let lines = wrap("prefix café 🦀 nextword end", 18, 2);
        assert_eq!(lines, vec!["prefix café 🦀", "  nextword end"]);
        let spans = highlight("café crab", "cf", Color::Cyan);
        assert_eq!(
            spans
                .iter()
                .filter(|s| s.style.add_modifier.contains(Modifier::UNDERLINED))
                .count(),
            2
        );
        let lines = diff_lines("- [ ] old\n", "- [x] old\n");
        assert!(
            lines
                .iter()
                .flat_map(|l| &l.spans)
                .any(|s| s.style.add_modifier.contains(Modifier::UNDERLINED))
        );
    }
}
