//! Windows console input, with independent key-down UTF-16 decoding.
//! Crossterm 0.29 combines down/up surrogate records, losing non-BMP input
//! when `ConPTY` emits high-down, high-up, low-down, low-up.
#![deny(clippy::as_conversions)]
use crossterm::event::{Event, KeyCode, KeyEvent, KeyModifiers};

fn wait_millis(timeout: std::time::Duration) -> u32 {
    // DWORD::MAX means INFINITE to WaitForSingleObject, not a long finite wait.
    u32::try_from(timeout.as_millis())
        .unwrap_or(u32::MAX - 1)
        .min(u32::MAX - 1)
}

#[derive(Default)]
pub struct Decoder {
    high: Option<u16>,
}

impl Decoder {
    pub fn key(&mut self, down: bool, virtual_key: u16, unit: u16, state: u32) -> Option<KeyEvent> {
        if !down {
            return None;
        }
        let mut modifiers = KeyModifiers::empty();
        if state & 0x10 != 0 {
            modifiers |= KeyModifiers::SHIFT;
        }
        if state & 0x0c != 0 {
            modifiers |= KeyModifiers::CONTROL;
        }
        if state & 0x03 != 0 {
            modifiers |= KeyModifiers::ALT;
        }
        // AltGr supplies translated text rather than an application shortcut.
        if state & 0x09 == 0x09 && unit != 0 {
            modifiers.remove(KeyModifiers::CONTROL | KeyModifiers::ALT);
        }
        let code = match virtual_key {
            0x08 => KeyCode::Backspace,
            0x09 if modifiers.contains(KeyModifiers::SHIFT) => KeyCode::BackTab,
            0x09 => KeyCode::Tab,
            0x0d => KeyCode::Enter,
            0x1b => KeyCode::Esc,
            0x21 => KeyCode::PageUp,
            0x22 => KeyCode::PageDown,
            0x23 => KeyCode::End,
            0x24 => KeyCode::Home,
            0x25 => KeyCode::Left,
            0x26 => KeyCode::Up,
            0x27 => KeyCode::Right,
            0x28 => KeyCode::Down,
            0x2d => KeyCode::Insert,
            0x2e => KeyCode::Delete,
            0x70..=0x87 => KeyCode::F(u8::try_from(virtual_key - 0x6f).ok()?),
            _ => {
                let ch = match unit {
                    0xd800..=0xdbff => {
                        self.high = Some(unit);
                        return None;
                    }
                    0xdc00..=0xdfff => {
                        char::decode_utf16([self.high.take()?, unit]).next()?.ok()?
                    }
                    1..=26 if modifiers.contains(KeyModifiers::CONTROL) => {
                        char::from_u32(u32::from(unit) + u32::from(b'a') - 1)?
                    }
                    0 => return None,
                    _ => char::from_u32(u32::from(unit))?,
                };
                KeyCode::Char(ch)
            }
        };
        self.high = None;
        Some(KeyEvent::new(code, modifiers))
    }
}

#[derive(Default)]
pub struct Input {
    pending: std::collections::VecDeque<Event>,
    escape: Vec<KeyEvent>,
    paste: Option<String>,
}

impl Input {
    fn key(&mut self, key: KeyEvent) {
        let ch = match key.code {
            KeyCode::Esc => Some('\x1b'),
            KeyCode::Enter => Some('\n'),
            KeyCode::Tab => Some('\t'),
            KeyCode::Char(ch) => Some(ch),
            _ => None,
        };
        if let Some(paste) = &mut self.paste {
            if let Some(ch) = ch {
                paste.push(ch);
                if paste.ends_with("\x1b[201~") {
                    paste.truncate(paste.len() - 6);
                    if let Some(paste) = self.paste.take() {
                        self.pending.push_back(Event::Paste(paste));
                    }
                }
            }
        } else if !self.escape.is_empty() || key.code == KeyCode::Esc {
            self.escape.push(key);
            let text: String = self
                .escape
                .iter()
                .map(|k| match k.code {
                    KeyCode::Esc => '\x1b',
                    KeyCode::Char(ch) => ch,
                    _ => '\0',
                })
                .collect();
            if text == "\x1b[200~" {
                self.escape.clear();
                self.paste = Some(String::new());
            } else if !"\x1b[200~".starts_with(&text) {
                self.flush_escape();
            }
        } else {
            self.pending.push_back(Event::Key(key));
        }
    }

    fn flush_escape(&mut self) {
        self.pending.extend(self.escape.drain(..).map(Event::Key));
    }
}

#[cfg(windows)]
#[derive(Default)]
pub struct Reader {
    decoder: Decoder,
    input: Input,
}

#[cfg(windows)]
impl Reader {
    pub fn next(
        &mut self,
        timeout: std::time::Duration,
    ) -> std::io::Result<Option<crossterm::event::Event>> {
        use crossterm::event::Event;
        use windows_sys::Win32::{
            Foundation::{INVALID_HANDLE_VALUE, WAIT_FAILED, WAIT_TIMEOUT},
            System::{
                Console::{
                    GetStdHandle, INPUT_RECORD, KEY_EVENT, ReadConsoleInputW, STD_INPUT_HANDLE,
                    WINDOW_BUFFER_SIZE_EVENT,
                },
                Threading::WaitForSingleObject,
            },
        };
        if let Some(event) = self.input.pending.pop_front() {
            return Ok(Some(event));
        }
        let deadline = std::time::Instant::now()
            .checked_add(timeout)
            .ok_or_else(|| {
                std::io::Error::new(
                    std::io::ErrorKind::InvalidInput,
                    "console timeout exceeds clock range",
                )
            })?;
        // SAFETY: STD_INPUT_HANDLE is a documented pseudo-handle selector. The
        // borrowed handle is neither retained beyond this call nor closed.
        let handle = unsafe { GetStdHandle(STD_INPUT_HANDLE) };
        if handle.is_null() || handle == INVALID_HANDLE_VALUE {
            return Err(std::io::Error::last_os_error());
        }
        loop {
            let remaining = deadline.saturating_duration_since(std::time::Instant::now());
            // SAFETY: handle is the live console input handle; timeout is bounded.
            match unsafe { WaitForSingleObject(handle, wait_millis(remaining)) } {
                WAIT_TIMEOUT => {
                    self.input.flush_escape();
                    return Ok(self.input.pending.pop_front());
                }
                WAIT_FAILED => return Err(std::io::Error::last_os_error()),
                _ => {}
            }
            let mut record = INPUT_RECORD::default();
            let mut count = 0;
            // SAFETY: both output pointers are valid and the buffer has one record.
            if unsafe { ReadConsoleInputW(handle, &raw mut record, 1, &raw mut count) } == 0 {
                return Err(std::io::Error::last_os_error());
            }
            if count != 0 {
                match u32::from(record.EventType) {
                    KEY_EVENT => {
                        // SAFETY: EventType identifies the active union member.
                        let key = unsafe { record.Event.KeyEvent };
                        // SAFETY: ReadConsoleInputW populates the Unicode member.
                        let unit = unsafe { key.uChar.UnicodeChar };
                        if let Some(event) = self.decoder.key(
                            key.bKeyDown != 0,
                            key.wVirtualKeyCode,
                            unit,
                            key.dwControlKeyState,
                        ) {
                            for _ in 0..key.wRepeatCount.max(1) {
                                self.input.key(event);
                            }
                        }
                    }
                    WINDOW_BUFFER_SIZE_EVENT => {
                        let (width, height) = crossterm::terminal::size()?;
                        self.input.pending.push_back(Event::Resize(width, height));
                    }
                    _ => {}
                }
            }
            if let Some(event) = self.input.pending.pop_front() {
                return Ok(Some(event));
            }
            if std::time::Instant::now() >= deadline {
                self.input.flush_escape();
                return Ok(self.input.pending.pop_front());
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn finite_wait_never_becomes_windows_infinite_sentinel() {
        use std::time::Duration;
        assert_eq!(wait_millis(Duration::ZERO), 0);
        assert_eq!(wait_millis(Duration::from_millis(50)), 50);
        assert_eq!(
            wait_millis(Duration::from_millis(u64::from(u32::MAX))),
            u32::MAX - 1
        );
        assert_eq!(wait_millis(Duration::MAX), u32::MAX - 1);
    }

    #[test]
    fn bracketed_paste_is_one_event_and_escape_still_cancels() {
        let mut input = Input::default();
        for ch in "\x1b[200~café 🦀\nsecond line\x1b[201~".chars() {
            let code = match ch {
                '\x1b' => KeyCode::Esc,
                '\n' => KeyCode::Enter,
                _ => KeyCode::Char(ch),
            };
            input.key(KeyEvent::new(code, KeyModifiers::empty()));
        }
        assert_eq!(
            input.pending.pop_front(),
            Some(Event::Paste("café 🦀\nsecond line".into()))
        );
        assert!(input.pending.is_empty());
        let esc = KeyEvent::new(KeyCode::Esc, KeyModifiers::empty());
        input.key(esc);
        input.flush_escape();
        assert_eq!(input.pending.pop_front(), Some(Event::Key(esc)));
    }

    #[test]
    fn conpty_surrogate_key_releases_do_not_drop_or_duplicate_emoji() {
        let mut d = Decoder::default();
        assert!(d.key(true, 0, 0xd83e, 0).is_none());
        assert!(d.key(false, 0, 0xd83e, 0).is_none());
        assert_eq!(d.key(true, 0, 0xdd80, 0).unwrap().code, KeyCode::Char('🦀'));
        assert!(d.key(false, 0, 0xdd80, 0).is_none());
        assert!(d.key(true, 0, 0xd83e, 0).is_none());
        assert_eq!(
            d.key(true, 0, u16::from(b'a'), 0).unwrap().code,
            KeyCode::Char('a')
        );
        assert!(d.key(true, 0, 0xdd80, 0).is_none());
    }

    #[test]
    fn translated_text_controls_navigation_and_altgr() {
        let mut d = Decoder::default();
        assert_eq!(d.key(true, 0, 0x00e9, 0).unwrap().code, KeyCode::Char('é'));
        assert_eq!(
            d.key(true, 0x44, 4, 8),
            Some(KeyEvent::new(KeyCode::Char('d'), KeyModifiers::CONTROL))
        );
        assert_eq!(d.key(true, 9, 9, 0x10).unwrap().code, KeyCode::BackTab);
        assert_eq!(d.key(true, 0x25, 0, 0).unwrap().code, KeyCode::Left);
        assert_eq!(
            d.key(true, 0x51, u16::from(b'@'), 9),
            Some(KeyEvent::new(KeyCode::Char('@'), KeyModifiers::empty()))
        );
    }
}
