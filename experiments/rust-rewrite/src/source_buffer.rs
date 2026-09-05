//! A private multiline draft. The cursor is always a UTF-8/CRLF boundary.
use crossterm::event::{KeyCode, KeyEvent, KeyModifiers};
use std::collections::VecDeque;

#[derive(Default)]
pub struct SourceBuffer {
    text: String,
    cursor: usize,
    selected: bool,
    undo: VecDeque<(String, usize)>,
}
impl SourceBuffer {
    pub fn new(text: String) -> Self {
        Self {
            text,
            ..Self::default()
        }
    }
    pub fn text(&self) -> &str {
        &self.text
    }
    pub const fn cursor(&self) -> usize {
        self.cursor
    }
    pub const fn selected(&self) -> bool {
        self.selected
    }
    fn remember(&mut self) {
        if self.undo.len() == 100 {
            self.undo.pop_front();
        }
        self.undo.push_back((self.text.clone(), self.cursor));
    }
    pub fn insert(&mut self, text: &str) {
        let text: String = text
            .chars()
            .filter(|c| !c.is_control() || matches!(c, '\n' | '\r' | '\t'))
            .collect();
        if text.is_empty() {
            return;
        }
        self.remember();
        if self.selected {
            self.text.clear();
            self.cursor = 0;
            self.selected = false;
        }
        self.text.insert_str(self.cursor, &text);
        self.cursor += text.len();
        self.normalize_cursor();
    }
    fn normalize_cursor(&mut self) {
        // Insertion/deletion can join a CR and LF that were previously separate.
        if self.text[..self.cursor].ends_with('\r') && self.text[self.cursor..].starts_with('\n') {
            self.cursor += 1;
        }
    }
    fn previous(&self) -> usize {
        if self.text[..self.cursor].ends_with("\r\n") {
            self.cursor - 2
        } else {
            self.text[..self.cursor]
                .char_indices()
                .next_back()
                .map_or(0, |(i, _)| i)
        }
    }
    fn next(&self) -> usize {
        if self.text[self.cursor..].starts_with("\r\n") {
            self.cursor + 2
        } else {
            self.cursor
                + self.text[self.cursor..]
                    .chars()
                    .next()
                    .map_or(0, char::len_utf8)
        }
    }
    pub fn line_position(&self) -> (usize, usize) {
        let before = &self.text[..self.cursor];
        (
            before.bytes().filter(|b| *b == b'\n').count(),
            self.cursor - before.rfind('\n').map_or(0, |i| i + 1),
        )
    }
    fn line_start(&self) -> usize {
        self.text[..self.cursor].rfind('\n').map_or(0, |i| i + 1)
    }
    fn line_end(&self) -> usize {
        let end = self.text[self.cursor..]
            .find('\n')
            .map_or(self.text.len(), |i| self.cursor + i);
        if end > 0 && self.text.as_bytes()[end - 1] == b'\r' {
            end - 1
        } else {
            end
        }
    }
    fn vertical(&mut self, down: bool) {
        let start = self.line_start();
        let column = self.text[start..self.cursor].chars().count();
        let target = if down {
            let Some(end) = self.text[self.cursor..].find('\n') else {
                return;
            };
            self.cursor + end + 1
        } else {
            if start == 0 {
                return;
            }
            self.text[..start - 1].rfind('\n').map_or(0, |i| i + 1)
        };
        let line = self.text[target..]
            .split('\n')
            .next()
            .unwrap_or("")
            .trim_end_matches('\r');
        self.cursor = target
            + line
                .char_indices()
                .nth(column)
                .map_or(line.len(), |(i, _)| i);
    }
    pub fn key(&mut self, key: KeyEvent) {
        let control = key.modifiers.contains(KeyModifiers::CONTROL);
        if control {
            match key.code {
                KeyCode::Char('a') => {
                    self.selected = true;
                    return;
                }
                KeyCode::Char('z') => {
                    if let Some((text, cursor)) = self.undo.pop_back() {
                        self.text = text;
                        self.cursor = cursor;
                        self.selected = false;
                    }
                    return;
                }
                KeyCode::Home => {
                    self.cursor = 0;
                    self.selected = false;
                    return;
                }
                KeyCode::End => {
                    self.cursor = self.text.len();
                    self.selected = false;
                    return;
                }
                _ => return,
            }
        }
        match key.code {
            KeyCode::Char(c)
                if !key
                    .modifiers
                    .intersects(KeyModifiers::ALT | KeyModifiers::SUPER) =>
            {
                self.insert(&c.to_string());
            }
            KeyCode::Enter => self.insert(if self.text.contains("\r\n") {
                "\r\n"
            } else {
                "\n"
            }),
            KeyCode::Tab => self.insert("    "),
            KeyCode::Backspace | KeyCode::Delete => {
                self.remember();
                if self.selected {
                    self.text.clear();
                    self.cursor = 0;
                } else if key.code == KeyCode::Backspace {
                    let start = self.previous();
                    self.text.replace_range(start..self.cursor, "");
                    self.cursor = start;
                } else {
                    self.text.replace_range(self.cursor..self.next(), "");
                }
                self.selected = false;
            }
            KeyCode::Left => {
                self.cursor = self.previous();
                self.selected = false;
            }
            KeyCode::Right => {
                self.cursor = self.next();
                self.selected = false;
            }
            KeyCode::Home => {
                self.cursor = self.line_start();
                self.selected = false;
            }
            KeyCode::End => {
                self.cursor = self.line_end();
                self.selected = false;
            }
            KeyCode::Up | KeyCode::Down => {
                self.vertical(key.code == KeyCode::Down);
                self.selected = false;
            }
            KeyCode::PageUp | KeyCode::PageDown => {
                for _ in 0..10 {
                    self.vertical(key.code == KeyCode::PageDown);
                }
                self.selected = false;
            }
            _ => {}
        }
        self.normalize_cursor();
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    fn key(b: &mut SourceBuffer, k: KeyCode) {
        b.key(KeyEvent::new(k, KeyModifiers::NONE));
    }
    #[test]
    fn unicode_crlf_edit_navigation_and_draft_undo() {
        let original = "é🦀\r\n界\r\n";
        let mut b = SourceBuffer::new(original.into());
        key(&mut b, KeyCode::End);
        key(&mut b, KeyCode::Right);
        assert_eq!(b.cursor(), "é🦀\r\n".len());
        key(&mut b, KeyCode::Backspace);
        assert_eq!(b.text(), "é🦀界\r\n");
        b.key(KeyEvent::new(KeyCode::Char('z'), KeyModifiers::CONTROL));
        assert_eq!(b.text(), original);
        key(&mut b, KeyCode::Up);
        key(&mut b, KeyCode::End);
        key(&mut b, KeyCode::Enter);
        assert_eq!(b.text(), "é🦀\r\n\r\n界\r\n");
        b.key(KeyEvent::new(KeyCode::Char('a'), KeyModifiers::CONTROL));
        b.insert("# New\n\n- [ ] café\n");
        assert_eq!(b.text(), "# New\n\n- [ ] café\n");
        let mut b = SourceBuffer::new("\n".into());
        b.insert("\r");
        assert_eq!((b.text(), b.cursor()), ("\r\n", 2));
        let mut b = SourceBuffer::new("\rx\n".into());
        key(&mut b, KeyCode::Right);
        key(&mut b, KeyCode::Delete);
        assert_eq!((b.text(), b.cursor()), ("\r\n", 2));
    }
    #[test]
    fn generated_edit_sequences_keep_utf8_cursor_valid() {
        let mut b = SourceBuffer::new("🦀 café\r\n界\n".into());
        let keys = [
            KeyCode::Left,
            KeyCode::Right,
            KeyCode::Up,
            KeyCode::Down,
            KeyCode::Home,
            KeyCode::End,
            KeyCode::Enter,
            KeyCode::Delete,
            KeyCode::Backspace,
            KeyCode::Char('界'),
        ];
        let mut seed = 11_u64;
        for _ in 0..1000 {
            seed = seed.wrapping_mul(6_364_136_223_846_793_005).wrapping_add(1);
            key(&mut b, keys[usize::try_from(seed % 10).unwrap()]);
            assert!(b.text().is_char_boundary(b.cursor()));
            assert!(
                !b.text()[..b.cursor()].ends_with('\r')
                    || !b.text()[b.cursor()..].starts_with('\n')
            );
        }
    }
}
