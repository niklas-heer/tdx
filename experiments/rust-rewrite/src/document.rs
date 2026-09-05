use crate::actions::Action;
use chrono::NaiveDate;
use pulldown_cmark::{Event, Options, Parser, Tag, TagEnd};
use regex::Regex;
use serde::{Deserialize, Serialize};
use std::{ops::Range, sync::LazyLock};

static TAG: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"#([a-zA-Z0-9_-]+)").unwrap());
static PRIORITY: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"!p(\d+)").unwrap());
static DUE: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"@due\((\d{4}-\d{2}-\d{2})\)").unwrap());

#[derive(Clone, Debug, Default, Deserialize, Serialize, PartialEq)]
#[serde(default, rename_all = "kebab-case")]
pub struct Metadata {
    pub read_only: Option<bool>,
    pub show_headings: Option<bool>,
    pub max_visible: Option<isize>,
    pub filter_done: Option<bool>,
    pub word_wrap: Option<bool>,
}
#[derive(Clone, Debug, Serialize, PartialEq)]
pub struct Task {
    pub index: usize,
    pub text: String,
    pub checked: bool,
    pub depth: usize,
    pub parent_index: Option<usize>,
    pub tags: Vec<String>,
    pub priority: i64,
    pub due_date: Option<String>,
    #[serde(skip)]
    pub marker: usize,
    #[serde(skip)]
    pub item: usize,
    #[serde(skip)]
    pub paragraph: Range<usize>,
}
#[derive(Clone, Debug, Serialize, PartialEq)]
pub struct Heading {
    pub level: usize,
    pub text: String,
    pub before_todo_index: usize,
    #[serde(skip)]
    pub range: Range<usize>,
    #[serde(skip)]
    pub prefix: String,
}
#[derive(Clone, Debug)]
pub struct Item {
    pub range: Range<usize>,
    pub list: usize,
    pub task: Option<usize>,
    pub prefix: String,
    pub marker: String,
    pub indent: usize,
}
#[derive(Clone, Debug)]
pub struct List {
    pub range: Range<usize>,
    pub parent: Option<usize>,
    pub items: Vec<usize>,
}
#[derive(Clone)]
pub struct Document {
    pub source: String,
    pub tasks: Vec<Task>,
    pub headings: Vec<Heading>,
    pub metadata: Metadata,
    pub items: Vec<Item>,
    pub lists: Vec<List>,
}
pub fn body_offset(source: &str) -> usize {
    let mut lines = source.split_inclusive('\n');
    if lines.next().is_none_or(|line| line.trim_end() != "---") {
        return 0;
    }
    let mut offset = source.find('\n').map_or(source.len(), |i| i + 1);
    for line in lines {
        offset += line.len();
        if line.trim_end() == "---" {
            return offset;
        }
    }
    0
}
fn line_start(source: &str, at: usize) -> usize {
    source.as_bytes()[..at]
        .iter()
        .rposition(|b| *b == b'\n')
        .map_or(0, |i| i + 1)
}
fn line_end(source: &str, at: usize) -> usize {
    source.as_bytes()[at..]
        .iter()
        .position(|b| *b == b'\n')
        .map_or(source.len(), |i| at + i + 1)
}
fn metadata(text: &str) -> (Vec<String>, i64, Option<String>) {
    let mut tags = Vec::new();
    for c in TAG.captures_iter(text) {
        if !tags.iter().any(|t| t == &c[1]) {
            tags.push(c[1].to_owned());
        }
    }
    let priority = PRIORITY
        .captures_iter(text)
        .filter_map(|c| c[1].parse::<i64>().ok())
        .filter(|p| *p > 0)
        .min()
        .unwrap_or(0);
    let due = DUE
        .captures_iter(text)
        .filter_map(|c| {
            NaiveDate::parse_from_str(&c[1], "%Y-%m-%d")
                .ok()
                .map(|_| c[1].to_owned())
        })
        .min();
    (tags, priority, due)
}
impl Document {
    pub fn parse(source: String) -> Result<Self, String> {
        let body = body_offset(&source);
        let meta = if body > 0 {
            let start = source.find('\n').unwrap() + 1;
            let end = source[..body].trim_end().rfind("---").unwrap();
            // Unknown frontmatter keys belong to the document and remain untouched.
            serde_yaml_ng::from_str::<Metadata>(&source[start..end]).unwrap_or_default()
        } else {
            Metadata::default()
        };
        let mut doc = Self {
            source: source.clone(),
            tasks: vec![],
            headings: vec![],
            metadata: meta,
            items: vec![],
            lists: vec![],
        };
        let mut list_stack: Vec<usize> = vec![];
        let mut item_stack: Vec<usize> = vec![];
        let mut paragraph = None;
        let mut skip_inline: usize = 0;
        let mut code_block = false;
        let mut primary: Vec<bool> = vec![];
        let options =
            Options::ENABLE_TASKLISTS | Options::ENABLE_TABLES | Options::ENABLE_STRIKETHROUGH;
        for (event, range) in Parser::new_ext(&source[body..], options).into_offset_iter() {
            let range = body + range.start..body + range.end;
            match &event {
                Event::Start(Tag::List(_)) => {
                    if let Some(i) = item_stack.last().and_then(|i| doc.items[*i].task) {
                        primary[i] = false;
                    }
                    let index = doc.lists.len();
                    doc.lists.push(List {
                        range: line_start(&source, range.start)
                            ..line_end(
                                &source,
                                range.start
                                    + source[range.clone()].trim_end().len().saturating_sub(1),
                            ),
                        parent: item_stack.last().copied(),
                        items: vec![],
                    });
                    list_stack.push(index);
                }
                Event::End(TagEnd::List(_)) => {
                    list_stack.pop();
                }
                Event::Start(Tag::Item) => {
                    let start = line_start(&source, range.start);
                    let marker_start = range.start
                        + source[range.start..]
                            .bytes()
                            .take_while(|b| *b == b' ' || *b == b'\t')
                            .count();
                    let prefix = source[start..marker_start].to_owned();
                    let tail = &source[marker_start..];
                    let width = tail.find(char::is_whitespace).unwrap_or(1);
                    let marker = tail[..width].to_owned();
                    let gap = tail[width..]
                        .chars()
                        .take_while(|c| *c == ' ' || *c == '\t')
                        .count()
                        .max(1);
                    let list = *list_stack.last().ok_or("item outside list")?;
                    let index = doc.items.len();
                    doc.items.push(Item {
                        range: start
                            ..line_end(
                                &source,
                                start + source[start..range.end].trim_end().len().saturating_sub(1),
                            ),
                        list,
                        task: None,
                        prefix,
                        marker,
                        indent: width + gap,
                    });
                    doc.lists[list].items.push(index);
                    item_stack.push(index);
                }
                Event::End(TagEnd::Item) => {
                    item_stack.pop();
                }
                Event::Start(Tag::Paragraph) => {
                    paragraph = Some(range.clone());
                }
                Event::End(TagEnd::Paragraph) => {
                    if let Some(i) = item_stack.last().and_then(|i| doc.items[*i].task) {
                        primary[i] = false;
                    }
                    paragraph = None;
                }
                Event::Start(Tag::Heading { level, .. }) => {
                    let start = line_start(&source, range.start);
                    let raw = source[range.clone()].trim_end();
                    let text = if raw.starts_with('#') {
                        raw.trim_start_matches('#')
                            .trim()
                            .trim_end_matches('#')
                            .trim()
                            .to_owned()
                    } else {
                        raw.lines().next().unwrap_or("").trim().to_owned()
                    };
                    doc.headings.push(Heading {
                        level: *level as usize,
                        text,
                        before_todo_index: doc.tasks.len(),
                        range: start..range.end,
                        prefix: source[start..range.start].into(),
                    });
                }
                Event::TaskListMarker(checked) => {
                    let item = *item_stack.last().ok_or("task outside item")?;
                    let list = doc.items[item].list;
                    let parent_index = doc.lists[list]
                        .parent
                        .and_then(|p| doc.items[p].task)
                        .map(|i| i + 1);
                    let index = doc.tasks.len();
                    doc.items[item].task = Some(index);
                    primary.push(true);
                    doc.tasks.push(Task {
                        index: index + 1,
                        text: String::new(),
                        checked: *checked,
                        depth: list_stack.len().saturating_sub(1),
                        parent_index,
                        tags: vec![],
                        priority: 0,
                        due_date: None,
                        marker: range.start,
                        item,
                        paragraph: paragraph
                            .clone()
                            .unwrap_or(range.start..line_end(&source, range.end)),
                    });
                }
                Event::Start(Tag::CodeBlock(_)) => {
                    if let Some(i) = item_stack.last().and_then(|i| doc.items[*i].task) {
                        primary[i] = false;
                    }
                    code_block = true;
                }
                Event::End(TagEnd::CodeBlock) => {
                    code_block = false;
                }
                _ => {}
            }
            let task = item_stack.last().and_then(|i| doc.items[*i].task);
            if let Some(index) = task {
                if primary[index]
                    && matches!(
                        event,
                        Event::Text(_)
                            | Event::Code(_)
                            | Event::InlineHtml(_)
                            | Event::SoftBreak
                            | Event::HardBreak
                            | Event::Start(Tag::Link { .. } | Tag::Image { .. })
                    )
                {
                    doc.tasks[index].paragraph.end = doc.tasks[index]
                        .paragraph
                        .end
                        .max(line_end(&source, range.end.saturating_sub(1)));
                }
                let text = &mut doc.tasks[index].text;
                match event {
                    Event::Start(Tag::Link {
                        dest_url,
                        link_type,
                        ..
                    }) => {
                        if skip_inline == 0 {
                            let raw = &source[range.clone()];
                            if matches!(
                                link_type,
                                pulldown_cmark::LinkType::Autolink
                                    | pulldown_cmark::LinkType::Email
                            ) {
                                text.push_str(raw);
                            } else {
                                let label = raw
                                    .strip_prefix('[')
                                    .and_then(|s| s.split_once(']').map(|p| p.0))
                                    .unwrap_or("");
                                let mut direct = String::new();
                                let mut depth = 0;
                                for e in Parser::new_ext(label, Options::ENABLE_STRIKETHROUGH) {
                                    match e {
                                        Event::Start(Tag::Paragraph) => {}
                                        Event::End(TagEnd::Paragraph) => {}
                                        Event::Start(_) => depth += 1,
                                        Event::End(_) => depth -= 1,
                                        Event::Text(t) if depth == 0 => direct.push_str(&t),
                                        _ => {}
                                    }
                                }
                                text.push_str(&format!("[{direct}]({dest_url})"));
                            }
                        }
                        skip_inline += 1;
                    }
                    Event::End(TagEnd::Link) => {
                        skip_inline = skip_inline.saturating_sub(1);
                    }
                    Event::Text(_) if skip_inline == 0 && !code_block => {
                        let start =
                            if range.start > 0 && source.as_bytes()[range.start - 1] == b'\\' {
                                range.start - 1
                            } else {
                                range.start
                            };
                        text.push_str(&source[start..range.end]);
                    }
                    Event::Code(value) if skip_inline == 0 => {
                        text.push('`');
                        text.push_str(&value);
                        text.push('`');
                    }
                    Event::InlineHtml(value) if skip_inline == 0 => {
                        text.push_str(&value);
                    }
                    Event::SoftBreak if skip_inline == 0 => {
                        text.push(' ');
                    }
                    _ => {}
                }
            }
        }
        for task in &mut doc.tasks {
            task.text = task.text.trim().into();
            (task.tags, task.priority, task.due_date) = metadata(&task.text);
        }
        Ok(doc)
    }
    pub fn query(&self) -> Result<&[Task], String> {
        Ok(&self.tasks)
    }
    pub fn section_end(&self, index: usize) -> usize {
        self.headings
            .iter()
            .skip(index + 1)
            .find(|h| h.level <= self.headings[index].level)
            .map_or(self.source.len(), |h| h.range.start)
    }
    pub fn section_bounds(&self, index: usize) -> Range<usize> {
        self.headings[index].before_todo_index
            ..self
                .headings
                .iter()
                .skip(index + 1)
                .find(|h| h.level <= self.headings[index].level)
                .map_or(self.tasks.len(), |h| h.before_todo_index)
    }
    pub fn change(&self, op: &str, index: usize, text: &str) -> Result<Self, String> {
        self.apply(&Action::new(op, index, text)).map(|(d, _)| d)
    }
    fn newline(&self) -> &str {
        if self.source.contains("\r\n") {
            "\r\n"
        } else {
            "\n"
        }
    }
    fn splice(&self, range: Range<usize>, text: &str) -> Result<Self, String> {
        let mut out = self.source.clone();
        out.replace_range(range, text);
        Self::parse(out)
    }
    fn task_item(&self, index: isize) -> Result<&Item, String> {
        self.tasks
            .get(usize::try_from(index).map_err(|_| "invalid task index")?)
            .map(|t| &self.items[t.item])
            .ok_or("invalid task index".into())
    }
    fn insert_text(
        &self,
        at: usize,
        prefix: &str,
        marker: &str,
        text: &str,
        checked: bool,
    ) -> Result<Self, String> {
        let nl = self.newline();
        let text = text.replace("\r\n", "\n");
        let mut lines = text.split('\n');
        let mut addition = format!(
            "{prefix}{marker} [{}] {}{nl}",
            if checked { "x" } else { " " },
            lines.next().unwrap_or("")
        );
        for line in lines {
            addition.push_str(&format!(
                "{prefix}{}{line}{nl}",
                " ".repeat(marker.len() + 1)
            ));
        }
        if at > 0 && !self.source[..at].ends_with('\n') {
            addition.insert_str(0, nl);
        }
        if at < self.source.len()
            && !self.source[at..]
                .trim_start_matches([' ', '\t'])
                .starts_with(['-', '+', '*', '\r', '\n'])
        {
            addition.push_str(nl);
        }
        self.splice(at..at, &addition)
    }
    fn relocated(&self, item: &Item, prefix: &str, marker: &str) -> String {
        let original = &self.source[item.range.clone()];
        let mut out = String::new();
        let mut lines = original.split_inclusive('\n');
        if let Some(line) = lines.next() {
            let tail = &line[item.prefix.len() + item.marker.len()..];
            out.push_str(prefix);
            out.push_str(marker);
            out.push_str(tail);
        }
        for line in lines {
            let rest = line.strip_prefix(&item.prefix).unwrap_or(line);
            out.push_str(prefix);
            let old = item.indent;
            let new = marker.len() + 1;
            if old > new {
                out.push_str(rest.strip_prefix(&" ".repeat(old - new)).unwrap_or(rest));
            } else {
                out.push_str(&" ".repeat(new - old));
                out.push_str(rest);
            }
        }
        if !out.ends_with('\n') {
            out.push_str(self.newline());
        }
        out
    }
    fn relocate(&self, item: &Item, at: usize, prefix: &str, marker: &str) -> Result<Self, String> {
        if at > item.range.start && at < item.range.end {
            return Err("cannot move a task into its own subtree".into());
        }
        let mut out = self.source.clone();
        let moved = self.relocated(item, prefix, marker);
        out.replace_range(item.range.clone(), "");
        let at = if at >= item.range.end {
            at - item.range.len()
        } else {
            at
        };
        out.insert_str(at, &moved);
        Self::parse(out)
    }
    pub fn apply(&self, a: &Action) -> Result<(Self, usize), String> {
        let (mut next, affected) = self.apply_raw(a)?;
        if (next.source != self.source
            || a.kind.starts_with("sort-")
            || (a.kind.starts_with("move") && a.index != a.target))
            && !matches!(
                a.kind.as_str(),
                "toggle" | "set-checked" | "set-all-checked" | "check-all" | "uncheck-all"
            )
        {
            let offset = body_offset(&next.source);
            if !next.source[offset..].trim_start().starts_with('#') {
                next.source.insert_str(offset, "# Todos\n\n");
                next = Self::parse(next.source)?;
            }
        }
        Ok((next, affected))
    }
    fn apply_raw(&self, a: &Action) -> Result<(Self, usize), String> {
        let index = usize::try_from(a.index).unwrap_or(0);
        if a.text.contains('\0') {
            return Err("task text contains a NUL character".into());
        }
        let next = match a.kind.as_str() {
            "toggle" | "set-checked" => {
                let t = self
                    .tasks
                    .get(index)
                    .filter(|_| a.index >= 0)
                    .ok_or("invalid task index")?;
                let checked = if a.kind == "toggle" {
                    !t.checked
                } else {
                    a.checked
                };
                self.splice(t.marker + 1..t.marker + 2, if checked { "x" } else { " " })?
            }
            "edit" => {
                let t = self
                    .tasks
                    .get(index)
                    .filter(|_| a.index >= 0)
                    .ok_or("invalid task index")?;
                let item = &self.items[t.item];
                let mut text = String::from(" ");
                let normalized = a.text.replace("\r\n", "\n");
                let mut lines = normalized.split('\n');
                text.push_str(lines.next().unwrap_or(""));
                for line in lines {
                    text.push_str(self.newline());
                    text.push_str(&item.prefix);
                    text.push_str(&" ".repeat(item.indent));
                    text.push_str(line);
                }
                text.push_str(self.newline());
                let end = t.paragraph.end.max(line_end(&self.source, t.marker));
                self.splice(t.marker + 3..end, &text)?
            }
            "delete" => {
                let item = self.task_item(a.index)?;
                let mut promoted = String::new();
                for list in self
                    .lists
                    .iter()
                    .filter(|l| l.parent == Some(self.tasks[index].item))
                {
                    for child in &list.items {
                        promoted.push_str(&self.relocated(
                            &self.items[*child],
                            &item.prefix,
                            &item.marker,
                        ));
                    }
                }
                self.splice(item.range.clone(), &promoted)?
            }
            "add" => {
                // The CLI and N append at document end, outside any prior nested list.
                let mut doc = self.clone();
                if !doc.source.is_empty() && !doc.source.ends_with('\n') {
                    doc.source.push_str(self.newline());
                }
                if !doc.source.is_empty()
                    && !doc.source.ends_with(&format!("{0}{0}", self.newline()))
                {
                    doc.source.push_str(self.newline());
                }
                for (event, range) in
                    Parser::new(&self.source[body_offset(&self.source)..]).into_offset_iter()
                {
                    if matches!(
                        event,
                        Event::Start(Tag::CodeBlock(pulldown_cmark::CodeBlockKind::Fenced(_)))
                    ) && range.end == self.source.len() - body_offset(&self.source)
                    {
                        let raw = &self.source[body_offset(&self.source) + range.start..];
                        let first = raw.lines().next().unwrap_or("").trim_start();
                        let fence: String = first
                            .chars()
                            .take_while(|c| *c == '`' || *c == '~')
                            .collect();
                        let closed = raw.lines().skip(1).any(|line| {
                            line.trim().starts_with(&fence)
                                && line.trim().chars().all(|c| fence.starts_with(c))
                        });
                        if !closed {
                            doc.source.push_str(&format!("{fence}\n\n"));
                        }
                    }
                }
                let at = doc.source.len();
                doc.insert_text(at, "", "-", &a.text, a.checked)?
            }
            "insert" => {
                let item = self.task_item(a.index)?;
                self.insert_text(
                    item.range.end,
                    &item.prefix,
                    &item.marker,
                    &a.text,
                    a.checked,
                )?
            }
            "add-in-section" => {
                let h = self
                    .headings
                    .get(index)
                    .filter(|_| a.index >= 0)
                    .ok_or("section no longer exists")?;
                let start = h.range.end;
                let immediate = self.lists.iter().find(|l| {
                    l.parent.is_none()
                        && l.range.start >= start
                        && self.source[start..l.range.start].trim().is_empty()
                });
                let at = immediate.map_or(start, |l| l.range.end);
                self.insert_text(at, &h.prefix, "-", &a.text, a.checked)?
            }
            "move" | "move-to-position" => {
                let item = self.task_item(a.index)?;
                let target = self.task_item(a.target)?;
                if target.range.start > item.range.start && target.range.start < item.range.end {
                    return Err("cannot move a task into its own subtree".into());
                }
                if a.index == a.target {
                    self.clone()
                } else {
                    let after = if a.kind == "move" {
                        a.index < a.target
                    } else {
                        a.insert_after
                    };
                    self.relocate(
                        item,
                        if after {
                            target.range.end
                        } else {
                            target.range.start
                        },
                        &target.prefix,
                        &target.marker,
                    )?
                }
            }
            "indent" => {
                let item = self.task_item(a.index)?;
                let list = &self.lists[item.list];
                let position = list
                    .items
                    .iter()
                    .position(|i| *i == self.tasks[index].item)
                    .unwrap();
                if position == 0 {
                    return Err("cannot indent: no previous sibling".into());
                }
                let previous = &self.items[list.items[position - 1]];
                let prefix = format!("{}{}", previous.prefix, " ".repeat(previous.indent));
                self.relocate(item, previous.range.end, &prefix, "-")?
            }
            "outdent" => {
                let item = self.task_item(a.index)?;
                let parent = self.lists[item.list]
                    .parent
                    .ok_or("cannot outdent: already at top level")?;
                let parent = &self.items[parent];
                self.relocate(item, parent.range.end, &parent.prefix, &parent.marker)?
            }
            "rename-heading" => {
                let h = self
                    .headings
                    .get(index)
                    .filter(|_| a.index >= 0)
                    .ok_or("section no longer exists")?;
                Self::valid_heading(&a.text, h.level)?;
                self.splice(
                    h.range.clone(),
                    &format!(
                        "{}{} {}{}",
                        h.prefix,
                        "#".repeat(h.level),
                        a.text.trim(),
                        self.newline()
                    ),
                )?
            }
            "create-heading" => {
                Self::valid_heading(&a.text, a.level)?;
                let (at, prefix) = if a.index == -1 {
                    (self.source.len(), "")
                } else {
                    let h = self.headings.get(index).ok_or("section no longer exists")?;
                    (self.section_end(index), h.prefix.as_str())
                };
                let nl = self.newline();
                self.splice(
                    at..at,
                    &format!(
                        "{nl}{prefix}{} {}{nl}{nl}",
                        "#".repeat(a.level),
                        a.text.trim()
                    ),
                )?
            }
            "check-all" | "uncheck-all" | "set-all-checked" => {
                let checked = if a.kind == "set-all-checked" {
                    a.checked
                } else {
                    a.kind == "check-all"
                };
                let mut source = self.source.clone();
                for task in &self.tasks {
                    source.replace_range(
                        task.marker + 1..task.marker + 2,
                        if checked { "x" } else { " " },
                    );
                }
                Self::parse(source)?
            }
            "clear-done" => {
                let mut doc = self.clone();
                for i in (0..self.tasks.len()).rev() {
                    if self.tasks[i].checked {
                        doc = doc.change("delete", i, "")?;
                    }
                }
                doc
            }
            "sort-done" | "sort-priority" | "sort-due" => self.sorted(&a.kind)?,
            _ => return Err(format!("unknown action {}", a.kind)),
        };
        let affected = match a.kind.as_str() {
            "add" => next.tasks.len().saturating_sub(1),
            "insert" => self
                .tasks
                .iter()
                .take_while(|t| {
                    self.items[t.item].range.start < self.items[self.tasks[index].item].range.end
                })
                .count(),
            "add-in-section" => next
                .tasks
                .iter()
                .rposition(|t| t.text == a.text)
                .unwrap_or(index),
            "create-heading" => next
                .headings
                .iter()
                .enumerate()
                .find(|(_, h)| {
                    h.text == a.text.trim()
                        && (a.index < 0 || h.range.start >= self.headings[index].range.end)
                })
                .map_or(0, |(i, _)| i),
            _ => index.min(next.tasks.len().saturating_sub(1)),
        };
        Ok((next, affected))
    }
    fn valid_heading(text: &str, level: usize) -> Result<(), String> {
        if text.trim().is_empty() || text.contains(['\r', '\n']) || !(1..=6).contains(&level) {
            Err("heading needs a single-line title and level 1–6".into())
        } else {
            Ok(())
        }
    }
    fn sorted(&self, kind: &str) -> Result<Self, String> {
        // Sort sibling task slots as whole subtrees; leave ordinary list items and prose in place.
        fn list_text(doc: &Document, list: usize, kind: &str) -> String {
            let list = &doc.lists[list];
            let mut blocks = Vec::new();
            for id in &list.items {
                let item = &doc.items[*id];
                let mut block = doc.source[item.range.clone()].to_owned();
                for (child_id, child) in doc
                    .lists
                    .iter()
                    .enumerate()
                    .rev()
                    .filter(|(_, l)| l.parent == Some(*id))
                {
                    block.replace_range(
                        child.range.start - item.range.start..child.range.end - item.range.start,
                        &list_text(doc, child_id, kind),
                    );
                }
                blocks.push((*id, block));
            }
            let mut tasks: Vec<_> = blocks
                .iter()
                .filter(|(i, _)| doc.items[*i].task.is_some())
                .cloned()
                .collect();
            tasks.sort_by_key(|(i, _)| {
                let t = &doc.tasks[doc.items[*i].task.unwrap()];
                match kind {
                    "sort-done" => (t.checked as i64, String::new()),
                    "sort-priority" => (
                        if t.priority == 0 {
                            i64::MAX
                        } else {
                            t.priority
                        },
                        String::new(),
                    ),
                    _ => (0, t.due_date.clone().unwrap_or("9999-99-99".into())),
                }
            });
            let mut sorted = tasks.into_iter();
            let mut out = doc.source[list.range.clone()].to_owned();
            for (i, block) in blocks.iter_mut() {
                if doc.items[*i].task.is_some() {
                    *block = sorted.next().unwrap().1;
                }
            }
            for (i, block) in blocks.into_iter().rev() {
                let r = &doc.items[i].range;
                out.replace_range(r.start - list.range.start..r.end - list.range.start, &block);
            }
            out
        }
        let mut out = self.source.clone();
        for (id, list) in self
            .lists
            .iter()
            .enumerate()
            .rev()
            .filter(|(_, l)| l.parent.is_none())
        {
            out.replace_range(list.range.clone(), &list_text(self, id, kind));
        }
        Self::parse(out)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn corpus_and_multiline_query() {
        let fixtures: serde_json::Value =
            serde_json::from_str(include_str!("../../rust-eval/fixtures.json")).unwrap();
        for fixture in fixtures.as_array().unwrap() {
            let source = fixture["source"].as_str().unwrap();
            let doc = Document::parse(source.into()).unwrap();
            let markers: Vec<_> = doc
                .tasks
                .iter()
                .map(|t| serde_json::json!({"checked":t.checked,"depth":t.depth}))
                .collect();
            assert_eq!(
                serde_json::json!(markers),
                fixture["markers"],
                "{}",
                fixture["name"]
            );
            for i in 0..doc.tasks.len() {
                let next = doc.change("toggle", i, "").unwrap();
                assert_eq!(source.len(), next.source.len());
                assert_eq!(
                    next.source
                        .bytes()
                        .zip(source.bytes())
                        .filter(|(a, b)| a != b)
                        .count(),
                    1
                );
            }
        }
        assert_eq!(
            Document::parse("- [ ] one\n  two\n".into()).unwrap().tasks[0].text,
            "one two"
        );
    }
    #[test]
    fn nested_delete_insert_and_relocate() {
        let doc = Document::parse(
            "- [ ] parent\n  - [ ] child\n    - [ ] grandchild\n- [ ] last\n".into(),
        )
        .unwrap();
        let next = doc.change("delete", 0, "").unwrap();
        assert_eq!(
            next.tasks.iter().map(|t| t.depth).collect::<Vec<_>>(),
            [0, 1, 0]
        );
        let (next, index) = doc.apply(&Action::new("insert", 0, "new")).unwrap();
        assert_eq!(index, 3);
        assert_eq!(next.tasks[index].text, "new");
        assert_eq!(next.tasks[index].depth, 0);
        let next = doc.change("indent", 3, "").unwrap();
        assert_eq!(next.tasks[3].depth, 1);
        let next = doc.change("outdent", 1, "").unwrap();
        assert_eq!(
            next.tasks.iter().map(|t| t.depth).collect::<Vec<_>>(),
            [0, 0, 1, 0]
        );
        assert!(
            doc.apply(&Action {
                kind: "move-to-position".into(),
                index: 0,
                target: 1,
                ..Action::default()
            })
            .is_err()
        );
    }
    #[test]
    fn sort_keeps_subtrees_prose_and_metadata() {
        let source = "---\ncustom: keep\n---\n# Project\n\nparagraph\n\n- [x] done\n  - [ ] child\n- [ ] open\n";
        let next = Document::parse(source.into())
            .unwrap()
            .change("sort-done", 0, "")
            .unwrap();
        assert_eq!(
            next.tasks
                .iter()
                .map(|t| t.text.as_str())
                .collect::<Vec<_>>(),
            ["open", "done", "child"]
        );
        assert!(
            next.source
                .starts_with("---\ncustom: keep\n---\n# Project\n\nparagraph\n")
        );
    }
}
