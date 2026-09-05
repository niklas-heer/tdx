//! Rendering helpers shared by the live terminal and buffer-level UI contracts.
use ratatui::{
    backend::{Backend, CrosstermBackend},
    buffer::Buffer,
    style::{Modifier, Style},
    text::{Line, Span},
};
use std::{
    collections::{BTreeMap, BTreeSet},
    io::{self, Write},
    sync::OnceLock,
};
use unicode_width::{UnicodeWidthChar, UnicodeWidthStr};

pub type Links = BTreeMap<(u16, u16), String>;
/// Keep a long path's filename visible without splitting Unicode glyphs.
pub fn tail(text: &str, width: usize) -> String {
    if text.width() <= width {
        return text.to_owned();
    }
    if width == 0 {
        return String::new();
    }
    let glyphs = glyphs(text, Style::default(), None);
    let mut used = 1;
    let mut start = glyphs.len();
    for (index, glyph) in glyphs.iter().enumerate().rev() {
        if used + glyph.width() > width {
            break;
        }
        used += glyph.width();
        start = index;
    }
    format!(
        "…{}",
        glyphs[start..]
            .iter()
            .map(|g| g.text.as_str())
            .collect::<String>()
    )
}
#[derive(Clone)]
pub struct Glyph {
    pub text: String,
    pub style: Style,
    pub url: Option<String>,
}
impl Glyph {
    fn width(&self) -> usize {
        self.text.width()
    }
}
pub fn glyphs(text: &str, style: Style, url: Option<&str>) -> Vec<Glyph> {
    let mut result: Vec<Glyph> = Vec::new();
    for ch in text.chars() {
        if ch.width() == Some(0)
            && let Some(last) = result.last_mut()
        {
            last.text.push(ch);
        } else {
            result.push(Glyph {
                text: ch.to_string(),
                style,
                url: url.map(str::to_owned),
            });
        }
    }
    result
}
pub fn inline(
    text: &str,
    base: Style,
    accent: Style,
    code: Style,
    plain: impl Fn(&str) -> Vec<Span<'static>>,
) -> Vec<Glyph> {
    static RE: OnceLock<regex::Regex> = OnceLock::new();
    let re = RE.get_or_init(|| regex::Regex::new(r"`([^`]+)`|\[([^\]]+)\]\(([^)]+)\)").unwrap());
    let mut out = Vec::new();
    let mut end = 0;
    for caps in re.captures_iter(text) {
        let whole = caps.get(0).unwrap();
        for span in plain(&text[end..whole.start()]) {
            out.extend(glyphs(&span.content, base.patch(span.style), None));
        }
        if let Some(body) = caps.get(1) {
            out.extend(glyphs(&format!(" {} ", body.as_str()), code, None));
        } else {
            let label = caps.get(2).unwrap().as_str();
            let url = caps.get(3).unwrap().as_str();
            let safe = !url.chars().any(char::is_control);
            out.extend(glyphs(
                label,
                accent.add_modifier(Modifier::UNDERLINED),
                safe.then_some(url),
            ));
        }
        end = whole.end();
    }
    for span in plain(&text[end..]) {
        out.extend(glyphs(&span.content, base.patch(span.style), None));
    }
    out
}
pub fn lines(
    mut input: Vec<Glyph>,
    width: usize,
    indent: usize,
    wrapping: bool,
) -> Vec<Vec<Glyph>> {
    if !wrapping || width < 4 {
        return vec![input];
    }
    let mut output = Vec::new();
    let padding = glyphs(&" ".repeat(indent.min(width / 2)), Style::default(), None);
    loop {
        let mut used = 0;
        let mut boundary = input.len();
        let mut space = None;
        for (i, g) in input.iter().enumerate() {
            if used + g.width() > width {
                boundary = i;
                break;
            }
            used += g.width();
            if g.text == " " && i > padding.len() {
                space = Some(i);
            }
        }
        if boundary == input.len() {
            output.push(input);
            break;
        }
        let cut = space.unwrap_or(boundary).max(1);
        let mut tail = input.split_off(cut);
        while tail.first().is_some_and(|g| g.text == " ") {
            tail.remove(0);
        }
        output.push(input);
        input = padding.iter().cloned().chain(tail).collect();
    }
    output
}
pub fn line(glyphs: &[Glyph]) -> Line<'static> {
    Line::from(
        glyphs
            .iter()
            .map(|g| Span::styled(g.text.clone(), g.style))
            .collect::<Vec<_>>(),
    )
}

/// A Unicode-safe horizontal input viewport, reserving a visible cursor cell.
pub fn input_window(text: &str, cursor: usize, width: usize) -> String {
    if width == 0 {
        return String::new();
    }
    let before = &text[..cursor];
    let after = &text[cursor..];
    let budget = width.saturating_sub(1);
    let mut used = 0;
    let mut start = cursor;
    for (i, c) in before.char_indices().rev() {
        let n = c.width().unwrap_or(0);
        if used + n > budget {
            break;
        }
        used += n;
        start = i;
    }
    let mut result = before[start..].to_owned();
    result.push('▏');
    for c in after.chars() {
        let n = c.width().unwrap_or(0);
        if used + n > budget {
            break;
        }
        used += n;
        result.push(c);
    }
    result
}

/// Ratatui's cells have no URL field. Repaint only current/former link cells with
/// OSC 8 metadata, using the normal backend for all colors and glyphs. Clear old
/// links even when Ratatui's glyph diff finds an unchanged cell after filtering.
pub fn sync_links(
    buffer: &Buffer,
    current: &Links,
    previous: &Links,
    writer: &mut impl Write,
) -> io::Result<()> {
    for &(x, y) in current
        .keys()
        .chain(previous.keys())
        .collect::<BTreeSet<_>>()
    {
        if !buffer.area.contains((x, y).into()) {
            continue;
        }
        let covered = (buffer.area.x..x)
            .rev()
            .take(2)
            .any(|p| p as usize + buffer[(p, y)].symbol().width() > x as usize);
        if covered {
            continue;
        }
        let url = current.get(&(x, y)).map(String::as_str).unwrap_or("");
        write!(writer, "\x1b]8;;{url}\x1b\\")?;
        CrosstermBackend::new(&mut *writer).draw(std::iter::once((x, y, &buffer[(x, y)])))?;
        write!(writer, "\x1b]8;;\x1b\\")?;
    }
    writer.flush()
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn long_unicode_input_keeps_cursor_and_tail_visible() {
        let text = "wide 界 café 🦀 final";
        let tail = input_window(text, text.len(), 10);
        assert!(tail.ends_with("final▏") && tail.width() <= 10);
        assert!(input_window(text, 0, 8).starts_with('▏'));
    }
    #[test]
    fn links_wrap_with_metadata_and_clear_without_glyph_changes() {
        let g = inline(
            "[long café link](https://example.com) `code`",
            Style::default(),
            Style::default(),
            Style::default(),
            |s| vec![Span::raw(s.to_owned())],
        );
        let rows = lines(g, 12, 0, true);
        assert!(rows.len() > 1 && rows[0][0].url.is_some());
        let mut buffer = Buffer::empty(ratatui::layout::Rect::new(0, 0, 12, 2));
        buffer.set_line(0, 0, &line(&rows[0]), 12);
        let links = Links::from([((0, 0), "https://example.com".into())]);
        let mut bytes = Vec::new();
        sync_links(&buffer, &links, &Links::new(), &mut bytes).unwrap();
        assert!(
            String::from_utf8(bytes)
                .unwrap()
                .contains("\x1b]8;;https://example.com\x1b\\")
        );
        let mut bytes = Vec::new();
        sync_links(&buffer, &Links::new(), &links, &mut bytes).unwrap();
        assert!(
            !String::from_utf8(bytes)
                .unwrap()
                .contains("https://example.com")
        );
    }
}
