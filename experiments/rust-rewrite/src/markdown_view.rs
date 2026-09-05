//! Render a document using Markdown events; HTML/control bytes are displayed as text.
use crate::presentation::{self, Glyph};
use pulldown_cmark::{Event, Options, Parser, Tag, TagEnd};
use ratatui::style::{Modifier, Style};

pub fn safe(text: &str) -> String {
    text.chars()
        .map(|c| {
            if c == '\t' {
                ' '
            } else if c.is_control() {
                '�'
            } else {
                c
            }
        })
        .collect()
}
struct Render {
    rows: Vec<Vec<Glyph>>,
    row: Vec<Glyph>,
    styles: Vec<Style>,
    links: Vec<String>,
    lists: Vec<Option<u64>>,
    base: Style,
    accent: Style,
    code: Style,
}
impl Render {
    fn push(&mut self, text: &str) {
        let style = self.styles.last().copied().unwrap_or(self.base);
        let url = self
            .links
            .last()
            .filter(|s| !s.chars().any(char::is_control))
            .map(String::as_str);
        self.row
            .extend(presentation::glyphs(&safe(text), style, url));
    }
    fn flush(&mut self) {
        self.rows.push(std::mem::take(&mut self.row));
    }
    fn boundary(&mut self) {
        if !self.row.is_empty() {
            self.flush();
        }
    }
    fn style(&mut self, style: Style) {
        self.styles.push(
            self.styles
                .last()
                .copied()
                .unwrap_or(self.base)
                .patch(style),
        );
    }
    #[expect(
        clippy::too_many_lines,
        reason = "Keep Markdown event rendering and paired tag handling together"
    )]
    fn event(&mut self, event: Event<'_>) {
        match event {
            Event::Start(tag) => match tag {
                Tag::Heading { .. } | Tag::TableHead => {
                    self.boundary();
                    self.style(self.accent.add_modifier(Modifier::BOLD));
                }
                Tag::Emphasis => self.style(Style::default().add_modifier(Modifier::ITALIC)),
                Tag::Strong => self.style(Style::default().add_modifier(Modifier::BOLD)),
                Tag::Strikethrough => {
                    self.style(Style::default().add_modifier(Modifier::CROSSED_OUT));
                }
                Tag::Link { dest_url, .. } => {
                    self.links.push(dest_url.into_string());
                    self.style(self.accent.add_modifier(Modifier::UNDERLINED));
                }
                Tag::Image { dest_url, .. } => {
                    self.push("[image: ");
                    self.links.push(dest_url.into_string());
                    self.style(self.accent);
                }
                Tag::CodeBlock(kind) => {
                    self.boundary();
                    self.style(self.code);
                    if let pulldown_cmark::CodeBlockKind::Fenced(language) = kind
                        && !language.is_empty()
                    {
                        self.push(&format!("┌ {language}"));
                        self.flush();
                    }
                }
                Tag::BlockQuote(_) => {
                    self.boundary();
                    self.push("│ ");
                }
                Tag::List(start) => {
                    self.boundary();
                    self.lists.push(start);
                }
                Tag::Item => {
                    self.boundary();
                    self.push(&"  ".repeat(self.lists.len().saturating_sub(1)));
                    let prefix = if let Some(Some(number)) = self.lists.last_mut() {
                        let value = format!("{number}. ");
                        *number = number.saturating_add(1);
                        value
                    } else {
                        "• ".into()
                    };
                    self.push(&prefix);
                }
                Tag::Table(_) | Tag::TableRow => self.boundary(),
                Tag::TableCell => self.push("│ "),
                _ => {}
            },
            Event::End(tag) => match tag {
                TagEnd::Heading(_) => {
                    self.styles.pop();
                    self.flush();
                    self.flush();
                }
                TagEnd::Paragraph => {
                    self.boundary();
                    if self.lists.is_empty() {
                        self.flush();
                    }
                }
                TagEnd::Emphasis | TagEnd::Strong | TagEnd::Strikethrough | TagEnd::Link => {
                    self.styles.pop();
                    if tag == TagEnd::Link {
                        self.links.pop();
                    }
                }
                TagEnd::Image => {
                    self.styles.pop();
                    self.links.pop();
                    self.push("]");
                }
                TagEnd::CodeBlock => {
                    self.styles.pop();
                    self.boundary();
                    self.flush();
                }
                TagEnd::Item | TagEnd::BlockQuote(_) => self.boundary(),
                TagEnd::List(_) => {
                    self.boundary();
                    self.lists.pop();
                    if self.lists.is_empty() {
                        self.flush();
                    }
                }
                TagEnd::TableHead => {
                    self.push("│");
                    self.flush();
                    self.styles.pop();
                }
                TagEnd::TableRow => {
                    self.push("│");
                    self.flush();
                }
                TagEnd::Table => {
                    self.boundary();
                    self.flush();
                }
                TagEnd::TableCell => self.push(" "),
                _ => {}
            },
            Event::Text(text) | Event::Html(text) | Event::InlineHtml(text) => {
                let mut lines = text.split('\n').peekable();
                while let Some(line) = lines.next() {
                    self.push(line.trim_end_matches('\r'));
                    if lines.peek().is_some() {
                        self.flush();
                    }
                }
            }
            Event::Code(text) => {
                self.style(self.code);
                self.push(&text);
                self.styles.pop();
            }
            Event::SoftBreak | Event::HardBreak => self.flush(),
            Event::Rule => {
                self.boundary();
                self.push("────────────────");
                self.flush();
            }
            Event::TaskListMarker(done) => self.push(if done { "☑ " } else { "☐ " }),
            Event::FootnoteReference(label) => self.push(&format!("[{label}]")),
            Event::InlineMath(text) | Event::DisplayMath(text) => self.push(&text),
        }
    }
}
pub fn render(source: &str, base: Style, accent: Style, code: Style) -> Vec<Vec<Glyph>> {
    let mut out = Render {
        rows: vec![],
        row: vec![],
        styles: vec![],
        links: vec![],
        lists: vec![],
        base,
        accent,
        code,
    };
    let body = crate::document::body_offset(source);
    if body > 0 {
        out.style(code);
        out.push("Frontmatter");
        out.flush();
        for line in source[..body].lines() {
            out.push(line);
            out.flush();
        }
        out.styles.pop();
        out.flush();
    }
    let options = Options::ENABLE_TABLES
        | Options::ENABLE_TASKLISTS
        | Options::ENABLE_STRIKETHROUGH
        | Options::ENABLE_FOOTNOTES;
    for event in Parser::new_ext(&source[body..], options) {
        out.event(event);
    }
    out.boundary();
    out.rows
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn complete_document_renders_structure_styles_links_and_safe_html() {
        let source = "---\ncustom: yes\n---\n# Heading\n\n**bold** and *italic* [link](https://example.com)\n\n- [ ] Task\n\n> Quote\n\n```rs\nlet x = 1;\n```\n\n| A | B |\n|---|---|\n| C | D |\n\n<div>opaque</div>\n";
        let rows = render(source, Style::default(), Style::default(), Style::default());
        let text = rows
            .iter()
            .map(|r| r.iter().map(|g| g.text.as_str()).collect::<String>())
            .collect::<Vec<_>>()
            .join("\n");
        for content in [
            "Frontmatter",
            "Heading",
            "bold",
            "italic",
            "☐ Task",
            "Quote",
            "let x = 1;",
            "│ A",
            "<div>opaque</div>",
        ] {
            assert!(text.contains(content), "missing {content}: {text}");
        }
        assert!(
            rows.iter()
                .flatten()
                .any(|g| g.style.add_modifier.contains(Modifier::BOLD))
        );
        assert!(
            rows.iter()
                .flatten()
                .any(|g| g.url.as_deref() == Some("https://example.com"))
        );
        assert!(!safe("hello\x1b[31m").contains('\x1b'));
    }
}
