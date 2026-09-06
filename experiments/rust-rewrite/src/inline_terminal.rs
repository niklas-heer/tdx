//! A content-sized region in the normal terminal buffer.
//!
//! Ratatui's automatic inline resize clears the entire screen when narrowing.
//! Fixed viewports let us instead reserve and clear only the rows we own.
use crossterm::{
    cursor::{Hide, MoveTo, Show},
    event::{DisableBracketedPaste, EnableBracketedPaste},
    execute,
    style::Print,
    terminal::{Clear, ClearType, disable_raw_mode, enable_raw_mode, size},
};
use ratatui::{Terminal, TerminalOptions, Viewport, backend::CrosstermBackend, layout::Rect};
use std::io::{self, Stdout};

struct Modes;
impl Drop for Modes {
    fn drop(&mut self) {
        let _ = execute!(io::stdout(), DisableBracketedPaste, Show);
        let _ = disable_raw_mode();
    }
}

pub struct InlineTerminal {
    pub terminal: Terminal<CrosstermBackend<Stdout>>,
    area: Rect,
    size: (u16, u16),
    _modes: Modes,
}
impl InlineTerminal {
    pub fn new() -> io::Result<Self> {
        enable_raw_mode()?;
        let modes = Modes;
        // Restore before the panic message is printed, including abort builds.
        let previous = std::panic::take_hook();
        std::panic::set_hook(Box::new(move |info| {
            let _ = execute!(io::stdout(), DisableBracketedPaste, Show, Print("\r\n"));
            let _ = disable_raw_mode();
            previous(info);
        }));
        execute!(io::stdout(), EnableBracketedPaste, Hide)?;
        let size = size()?;
        let (x, mut y) = crossterm::cursor::position()?;
        if x != 0 {
            execute!(io::stdout(), Print("\r\n"))?;
            y = y.saturating_add(1).min(size.1.saturating_sub(1));
        }
        let area = Rect::new(0, y, size.0, 1);
        let terminal = Self::fixed(area)?;
        Ok(Self {
            terminal,
            area,
            size,
            _modes: modes,
        })
    }
    fn fixed(area: Rect) -> io::Result<Terminal<CrosstermBackend<Stdout>>> {
        Terminal::with_options(
            CrosstermBackend::new(io::stdout()),
            TerminalOptions {
                viewport: Viewport::Fixed(area),
            },
        )
    }

    /// Rebuild buffers instead of calling Ratatui's screen-clearing resize.
    pub fn prepare(&mut self, width: u16, height: u16, rows: u16) -> io::Result<bool> {
        let rows = rows.clamp(1, height.max(1));
        if self.size == (width, height) && self.area.height == rows {
            return Ok(false);
        }
        let old_height = self.area.height.min(height);
        // The real cursor is parked on our last row. Query it after a terminal
        // resize because reflow of earlier shell output can move the region.
        let origin = if self.size == (width, height) {
            self.area.y
        } else {
            crossterm::cursor::position()?
                .1
                .saturating_sub(old_height.saturating_sub(1))
        }
        .min(height.saturating_sub(old_height));
        for y in origin..origin.saturating_add(old_height) {
            execute!(io::stdout(), MoveTo(0, y), Clear(ClearType::CurrentLine))?;
        }
        if rows > old_height {
            execute!(
                io::stdout(),
                MoveTo(0, origin.saturating_add(old_height).saturating_sub(1)),
                Print("\r\n".repeat(usize::from(rows - old_height)))
            )?;
        }
        let y = origin.min(height.saturating_sub(rows));
        self.area = Rect::new(0, y, width, rows);
        self.size = (width, height);
        self.terminal = Self::fixed(self.area)?;
        execute!(io::stdout(), Hide)?;
        Ok(true)
    }
    /// Keep the real cursor below the checklist; the editor paints its own caret.
    pub fn park_cursor(&self) -> io::Result<()> {
        execute!(
            io::stdout(),
            MoveTo(0, self.area.bottom().saturating_sub(1)),
            Hide
        )
    }
}
impl Drop for InlineTerminal {
    fn drop(&mut self) {
        if !std::thread::panicking() {
            let _ = execute!(
                io::stdout(),
                MoveTo(0, self.area.bottom().saturating_sub(1)),
                Print("\r\n")
            );
        }
    }
}
