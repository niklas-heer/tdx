use chrono::NaiveDate;
use pulldown_cmark::{Event, Options, Parser, Tag, TagEnd};
use regex::Regex;
use serde::Serialize;
use std::{ops::Range, sync::LazyLock};

static TAG: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"#([a-zA-Z0-9_-]+)").unwrap());
static PRIORITY: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"!p(\d+)").unwrap());
static DUE: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"@due\((\d{4}-\d{2}-\d{2})\)").unwrap());

// Match Go's query convention: emphasis delimiters are omitted, while inline
// links/code/HTML stay as Markdown. This is deliberately not a general renderer.
fn task_text(raw: &str) -> String {
    if !raw.contains(['*', '_', '~']) {
        return raw.to_owned();
    }
    let mut removals = Vec::new();
    for (event, range) in Parser::new_ext(raw, Options::ENABLE_STRIKETHROUGH).into_offset_iter() {
        let width = match event {
            Event::Start(Tag::Emphasis) => 1,
            Event::Start(Tag::Strong | Tag::Strikethrough) => 2,
            _ => continue,
        };
        removals.push(range.start..range.start + width);
        removals.push(range.end - width..range.end);
    }
    removals.sort_by_key(|r| std::cmp::Reverse(r.start));
    let mut text = raw.to_owned();
    for range in removals {
        text.replace_range(range, "");
    }
    text
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
    line: Range<usize>,
    #[serde(skip)]
    item_end: usize,
    #[serde(skip)]
    own_end: usize,
}

#[derive(Clone)]
pub struct Document {
    pub source: String,
    pub tasks: Vec<Task>,
    pub readonly: bool,
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

impl Document {
    pub fn parse(source: String) -> Result<Self, String> {
        let body = body_offset(&source);
        let readonly = if body > 0 {
            let first = source.find('\n').unwrap() + 1;
            let end = source[..body].trim_end().rfind("---").unwrap();
            let yaml: serde_yaml_ng::Value = serde_yaml_ng::from_str(&source[first..end])
                .map_err(|e| format!("invalid frontmatter: {e}"))?;
            match yaml.get("read-only") {
                Some(value) => value
                    .as_bool()
                    .ok_or("read-only frontmatter must be boolean")?,
                None => false,
            }
        } else {
            false
        };
        let options =
            Options::ENABLE_TASKLISTS | Options::ENABLE_TABLES | Options::ENABLE_STRIKETHROUGH;
        let mut tasks: Vec<Task> = Vec::new();
        let mut depth: usize = 0;
        let mut items: Vec<(usize, Option<usize>)> = Vec::new();
        for (event, range) in Parser::new_ext(&source[body..], options).into_offset_iter() {
            match event {
                Event::Start(Tag::List(_)) => {
                    if let Some((_, Some(index))) = items.last() {
                        tasks[*index].own_end = tasks[*index].own_end.min(body + range.start);
                    }
                    depth += 1;
                }
                Event::End(TagEnd::List(_)) => depth = depth.saturating_sub(1),
                Event::Start(Tag::Item) => items.push((body + range.end, None)),
                Event::End(TagEnd::Item) => {
                    items.pop();
                }
                Event::TaskListMarker(checked) => {
                    let marker = body + range.start;
                    let line_start = source[..marker].rfind('\n').map_or(0, |i| i + 1);
                    let line_end = source[marker..]
                        .find('\n')
                        .map_or(source.len(), |i| marker + i + 1);
                    let text = task_text(source[marker + 3..line_end].trim());
                    let mut tags = Vec::new();
                    for capture in TAG.captures_iter(&text) {
                        if !tags.iter().any(|tag| tag == &capture[1]) {
                            tags.push(capture[1].to_owned());
                        }
                    }
                    let priority = PRIORITY
                        .captures_iter(&text)
                        .filter_map(|c| c[1].parse::<i64>().ok())
                        .filter(|p| *p > 0)
                        .min()
                        .unwrap_or(0);
                    let due_date = DUE
                        .captures_iter(&text)
                        .filter_map(|c| {
                            NaiveDate::parse_from_str(&c[1], "%Y-%m-%d")
                                .ok()
                                .map(|_| c[1].to_owned())
                        })
                        .min();
                    let parent_index = items.iter().rev().find_map(|(_, task)| task.map(|i| i + 1));
                    let index = tasks.len();
                    let end = items.last().ok_or("task outside list item")?.0;
                    tasks.push(Task {
                        index: index + 1,
                        text,
                        checked,
                        depth: depth.saturating_sub(1),
                        parent_index,
                        tags,
                        priority,
                        due_date,
                        marker,
                        line: line_start..line_end,
                        item_end: end,
                        own_end: end,
                    });
                    items.last_mut().unwrap().1 = Some(index);
                }
                _ => {}
            }
        }
        Ok(Self {
            source,
            tasks,
            readonly,
        })
    }

    pub fn query(&self) -> Result<&[Task], String> {
        for task in &self.tasks {
            if task.own_end > task.line.end
                && !self.source[task.line.end..task.own_end]
                    .trim_matches(|c: char| c.is_whitespace() || c == '>')
                    .is_empty()
            {
                return Err("prototype cannot query multiline task bodies yet".into());
            }
        }
        Ok(&self.tasks)
    }

    pub fn change(&self, op: &str, index: usize, text: &str) -> Result<Self, String> {
        if self.readonly {
            return Err("read-only frontmatter: editing is disabled".into());
        }
        let mut output = self.source.clone();
        if (op == "add" || op == "edit")
            && (text.trim().is_empty() || text.chars().any(char::is_control))
        {
            return Err(
                "task text must be non-empty and on one line without control characters".into(),
            );
        }
        match op {
            "toggle" => {
                let task = self.tasks.get(index).ok_or("invalid task index")?;
                output.replace_range(
                    task.marker + 1..task.marker + 2,
                    if task.checked { " " } else { "x" },
                );
            }
            "edit" | "delete" => {
                let task = self.tasks.get(index).ok_or("invalid task index")?;
                // An item can include paragraphs, children, fences or quotes. Only
                // edit a single-line label; removal also requires no children.
                let end = if op == "edit" {
                    task.own_end
                } else {
                    task.item_end
                };
                if end > task.line.end
                    && !self.source[task.line.end..end]
                        .trim_matches(|c: char| c.is_whitespace() || c == '>')
                        .is_empty()
                {
                    return Err(
                        "prototype cannot delete parents or edit multiline task bodies".into(),
                    );
                }
                if op == "delete" {
                    output.replace_range(task.line.clone(), "");
                } else {
                    let end = task.line.end
                        - if self.source[..task.line.end].ends_with("\r\n") {
                            2
                        } else if self.source[..task.line.end].ends_with('\n') {
                            1
                        } else {
                            0
                        };
                    output.replace_range(task.marker + 3..end, &format!(" {}", text.trim()));
                }
            }
            "add" => {
                // Appending after an unclosed fence/HTML block would hide a task.
                // Validate the reparsed result below before any write.
                let newline = if self.source.contains("\r\n") {
                    "\r\n"
                } else {
                    "\n"
                };
                if !output.is_empty() && !output.ends_with('\n') {
                    output.push_str(newline);
                }
                if self.tasks.last().is_none_or(|task| {
                    task.depth != 0 || !self.source[task.line.end..].trim().is_empty()
                }) {
                    output.push_str(newline);
                }
                output.push_str(&format!("- [ ] {}{newline}", text.trim()));
            }
            _ => return Err(format!("unsupported edit: {op}")),
        }
        let next = Self::parse(output)?;
        let mut expected: Vec<(String, bool, usize)> = self
            .tasks
            .iter()
            .map(|t| (t.text.clone(), t.checked, t.depth))
            .collect();
        match op {
            "toggle" => expected[index].1 = !expected[index].1,
            "edit" => expected[index].0 = task_text(text.trim()),
            "delete" => {
                expected.remove(index);
            }
            "add" => expected.push((task_text(text.trim()), false, 0)),
            _ => unreachable!(),
        }
        let actual: Vec<_> = next
            .tasks
            .iter()
            .map(|t| (t.text.clone(), t.checked, t.depth))
            .collect();
        if actual != expected {
            return Err("unsupported Markdown structure: edit would change other tasks".into());
        }
        Ok(next)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn shared_markers_and_byte_exact_toggles() {
        let fixtures: serde_json::Value =
            serde_json::from_str(include_str!("../../rust-eval/fixtures.json")).unwrap();
        for fixture in fixtures.as_array().unwrap() {
            let source = fixture["source"].as_str().unwrap();
            let mut doc = Document::parse(source.to_owned()).unwrap();
            let markers: Vec<_> = doc
                .tasks
                .iter()
                .map(|task| serde_json::json!({"checked":task.checked,"depth":task.depth}))
                .collect();
            assert_eq!(
                serde_json::json!(markers),
                fixture["markers"],
                "{}",
                fixture["name"]
            );
            // Parser corpus tests mutation mechanics separately from read-only policy.
            doc.readonly = false;
            for index in 0..doc.tasks.len() {
                let next = doc.change("toggle", index, "").unwrap();
                assert_eq!(next.source.len(), source.len());
                assert_eq!(
                    next.source
                        .bytes()
                        .zip(source.bytes())
                        .filter(|(a, b)| a != b)
                        .count(),
                    1
                );
            }
            assert!(doc.change("toggle", doc.tasks.len(), "").is_err());
        }
    }
    #[test]
    fn metadata_duplicates_and_calendar_validation() {
        let doc = Document::parse("- [ ] café #tag #tag #other !p0 !p10 !p2 @due(2027-02-29) @due(2028-02-29) @due(2026-12-31)\n".into()).unwrap();
        let task = &doc.tasks[0];
        assert_eq!(task.tags, ["tag", "other"]);
        assert_eq!(task.priority, 2);
        assert_eq!(task.due_date.as_deref(), Some("2026-12-31"));
    }
    #[test]
    fn structural_rejections_and_line_endings() {
        for source in [
            "- [ ] parent\n  - [ ] child\n",
            "- [ ] paragraph\n  continuation\n",
        ] {
            let doc = Document::parse(source.into()).unwrap();
            assert!(doc.change("delete", 0, "").is_err());
            if source.contains("continuation") {
                assert!(doc.change("edit", 0, "new").is_err());
            } else {
                assert_eq!(doc.change("edit", 0, "new").unwrap().tasks[1].text, "child");
            }
        }
        let doc = Document::parse("- [ ] paragraph\n  continuation\n".into()).unwrap();
        assert!(doc.query().is_err());
        let doc = Document::parse("```md\nexample\n".into()).unwrap();
        assert!(doc.change("add", 0, "new").is_err());
        let doc = Document::parse("# Title\r\n\r\n- [ ] old\r\n".into()).unwrap();
        assert_eq!(
            doc.change("edit", 0, "新しい").unwrap().source,
            "# Title\r\n\r\n- [ ] 新しい\r\n"
        );
        assert_eq!(
            doc.change("delete", 0, "").unwrap().source,
            "# Title\r\n\r\n"
        );
        assert!(doc.change("add", 0, "injected\n- [ ] task").is_err());
    }
    #[test]
    fn frontmatter_policy_and_parent_mapping() {
        let doc = Document::parse(
            "---\nread-only: true\nunknown: keep\n---\n- [ ] parent\n  - [ ] child\n".into(),
        )
        .unwrap();
        assert!(doc.change("toggle", 0, "").is_err());
        assert_eq!(doc.tasks[1].parent_index, Some(1));
        assert!(Document::parse("---\nread-only: maybe\n---\n".into()).is_err());
    }
}
