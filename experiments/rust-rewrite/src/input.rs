use crossterm::event::{KeyCode, KeyEvent, KeyModifiers};
#[derive(Default, Clone, Debug)]
pub struct Buffer {
    // Only the editing methods can change these together, so every cursor is
    // an in-bounds UTF-8 boundary. Callers receive read-only views.
    text: String,
    cursor: usize,
}
impl Buffer {
    pub fn text(&self) -> &str {
        &self.text
    }
    pub const fn cursor(&self) -> usize {
        self.cursor
    }
    pub const fn new(text: String) -> Self {
        let cursor = text.len();
        Self { text, cursor }
    }
    pub fn insert(&mut self, text: &str) {
        let text = single_line(text);
        self.text.insert_str(self.cursor, &text);
        self.cursor += text.len();
    }
    pub fn key(&mut self, key: KeyEvent) -> bool {
        match key.code {
            KeyCode::Left => {
                self.cursor = self.text[..self.cursor]
                    .char_indices()
                    .last()
                    .map_or(0, |(i, _)| i);
            }
            KeyCode::Right => {
                self.cursor += self.text[self.cursor..]
                    .chars()
                    .next()
                    .map_or(0, char::len_utf8);
            }
            KeyCode::Home => self.cursor = 0,
            KeyCode::End => self.cursor = self.text.len(),
            KeyCode::Backspace => {
                let previous = self.text[..self.cursor]
                    .char_indices()
                    .last()
                    .map_or(0, |(i, _)| i);
                self.text.replace_range(previous..self.cursor, "");
                self.cursor = previous;
            }
            KeyCode::Delete => {
                let end = self.cursor
                    + self.text[self.cursor..]
                        .chars()
                        .next()
                        .map_or(0, char::len_utf8);
                self.text.replace_range(self.cursor..end, "");
            }
            KeyCode::Char('a') if key.modifiers.contains(KeyModifiers::CONTROL) => self.cursor = 0,
            KeyCode::Char('e') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                self.cursor = self.text.len();
            }
            KeyCode::Char('u') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                self.text.clear();
                self.cursor = 0;
            }
            KeyCode::Char('h') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                return self.key(KeyEvent::new(KeyCode::Backspace, KeyModifiers::NONE));
            }
            KeyCode::Char(c)
                if !key.modifiers.intersects(
                    KeyModifiers::CONTROL | KeyModifiers::ALT | KeyModifiers::SUPER,
                ) =>
            {
                self.insert(&c.to_string());
            }
            _ => return false,
        }
        true
    }
}
pub fn single_line(s: &str) -> String {
    s.split(['\n', '\r'])
        .next()
        .unwrap_or("")
        .chars()
        .filter(|c| !c.is_control())
        .collect()
}
pub fn fuzzy(query: &str, text: &str) -> usize {
    let query = query.to_lowercase();
    let text = text.to_lowercase();
    if query.is_empty() {
        return 1;
    }
    if text.contains(&query) {
        return 1000 + query.len();
    }
    let mut query = query.chars().peekable();
    let mut score = 0;
    let mut previous = None;
    let mut prior = ' ';
    for (i, c) in text.chars().enumerate() {
        if query.peek() == Some(&c) {
            score += 10;
            if previous == i.checked_sub(1) {
                score += 5;
            }
            if i == 0 || prior == ' ' {
                score += 3;
            }
            previous = Some(i);
            query.next();
        }
        prior = c;
    }
    if query.peek().is_none() { score } else { 0 }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn unicode_cursor_delete_and_paste() {
        let mut b = Buffer::new("café🦀Z".into());
        b.key(KeyEvent::new(KeyCode::Left, KeyModifiers::NONE));
        b.key(KeyEvent::new(KeyCode::Backspace, KeyModifiers::NONE));
        assert_eq!(b.text, "caféZ");
        b.key(KeyEvent::new(KeyCode::Left, KeyModifiers::NONE));
        b.key(KeyEvent::new(KeyCode::Delete, KeyModifiers::NONE));
        assert_eq!(b.text, "cafZ");
        b.insert("🦀\nignored");
        assert_eq!(b.text, "caf🦀Z");
        assert!(b.text.is_char_boundary(b.cursor));
    }

    #[test]
    fn mixed_unicode_edit_sequences_preserve_cursor_invariants() {
        let mut buffer = Buffer::new("café 🦀 界 e\u{301}".into());
        let keys = [
            KeyCode::Left,
            KeyCode::Right,
            KeyCode::Home,
            KeyCode::End,
            KeyCode::Backspace,
            KeyCode::Delete,
            KeyCode::Char('界'),
            KeyCode::Char('🦀'),
        ];
        let mut seed = 17_u64;
        for step in 0..256 {
            seed = seed.wrapping_mul(6_364_136_223_846_793_005).wrapping_add(1);
            let index = usize::try_from(seed % 8).unwrap();
            buffer.key(KeyEvent::new(keys[index], KeyModifiers::NONE));
            if step % 11 == 0 {
                buffer.insert("é\nignored");
            }
            assert!(buffer.text().is_char_boundary(buffer.cursor()));
            assert!(!buffer.text().contains('\n'));
        }
    }
}
