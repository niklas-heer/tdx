"""Exercise the real process while a VT emulator tracks prior terminal context.

Requires pyte==0.8.2 and wcwidth==0.8.3 (test dependencies only).
"""
import argparse
import json
import os
from pathlib import Path
import tempfile

import pyte
from native_check import terminal_type


class Screen(pyte.HistoryScreen):
    def resize(self, lines=None, columns=None):
        # pyte always deletes from the top when shrinking. Real normal-buffer
        # terminals first remove unused rows below the cursor. Model that here
        # so a short checklist is not artificially shifted by the test adapter.
        if lines is not None and lines < self.lines:
            trimmed = min(self.lines - lines, self.lines - self.cursor.y - 1)
            for y in range(self.lines - trimmed, self.lines):
                self.buffer.pop(y, None)
            self.lines -= trimmed
        super().resize(lines=lines, columns=columns)


def check(binary, output):
    output.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='tdx-inline-') as tmp:
        base = Path(tmp)
        path = base / 'tasks.md'
        original = '- [ ] Alpha\n- [ ] Beta\n'
        path.write_text(original)
        screen = Screen(100, 32, history=1000)
        context = 'previous command\r\nprevious result\r\n$ tdx tasks.md\r\n'
        term = terminal_type()(binary, path, base / 'config', context=context, screen=screen)
        def row(content):
            return next((i for i, line in enumerate(screen.display) if content in line), None)
        def shown(): return '\n'.join(screen.display)
        def preserved():
            assert screen.display[:3] == [s.ljust(screen.columns) for s in context.split('\r\n')[:3]], shown()
        def capture(name): (output / f'inline-{name}.txt').write_text(shown()+'\n')
        try:
            term.until(lambda: row('Alpha') is not None and row('Esc quit') is not None, 'compact initial checklist')
            preserved()
            assert row('Alpha') == 3 and row('Beta') == 4, shown()
            assert max(i for i, line in enumerate(screen.display) if line.strip()) < 11, shown()
            capture('tasks')
            term.send(b'nInline draft')
            term.until(lambda: 'Inline draft▏' in shown(), 'inline new task')
            assert row('Alpha') + 1 == row('Inline draft▏') == row('Beta') - 1, shown()
            preserved(); capture('new')
            assert path.read_text() == original
            term.send(b'\x1b'); term.pump(.65)
            term.until(lambda: row('Inline draft') is None, 'cancel removes draft row')
            assert path.read_text() == original
            term.send('e café 🦀'.encode())
            term.until(lambda: 'café 🦀▏' in shown(), 'edit replaces original row')
            assert row('café 🦀▏') == 3 and row('Beta') == 4, shown()
            capture('edit'); term.send(b'\r')
            term.until(lambda: 'Alpha café 🦀' in path.read_text(), 'save inline edit')
            term.send(b'u'); term.until(lambda: path.read_text() == original, 'undo inline edit')
            term.send(b'NAppended')
            term.until(lambda: 'Appended▏' in shown(), 'append editor')
            assert row('Appended▏') == row('Beta') + 1, shown()
            term.send(b'\r'); term.until(lambda: 'Appended' in path.read_text(), 'save appended task')
            term.send(b'u'); term.until(lambda: path.read_text() == original, 'undo append')
            for width, height in [(40, 32), (120, 32), (80, 20), (100, 32)]:
                term.resize(width, height); term.send(b'g')
                term.pump(.3)
                term.until(lambda: row('Alpha') is not None and row('Beta') is not None, 'resize keeps tasks')
                preserved()
            capture('resized')
            term.send(b'?'); term.until(lambda: 'Help · Home/End' in shown(), 'open help')
            preserved()
            term.send(b'\x1b'); term.pump(.65)
            term.until(lambda: row('Alpha') == 3 and row('Beta') == 4 and row('Help · Home/End') is None, 'shrink after help')
            preserved()
            assert all(not line.strip() for line in screen.display[11:]), shown()
            term.send(b':markdown\r'); term.until(lambda: 'Markdown source' in shown(), 'full source tool')
            term.send(b'\x1b'); term.pump(.65)
            term.until(lambda: row('Alpha') == 3 and row('Markdown source') is None, 'shrink after Markdown')
            preserved()
            term.close(); preserved(); capture('closed')
            assert not screen.cursor.hidden
            if os.name != 'nt':
                import termios
                restored = termios.tcgetattr(term.fd)[3]
                assert restored & termios.ICANON and restored & termios.ECHO
            assert path.read_text() == original
            # ConPTY may emit its own initial screen-clear while projecting the
            # console. On Unix these bytes come directly from the application.
            if os.name != 'nt':
                for sequence in (b'\x1b[?1049h', b'\x1b[?47h', b'\x1b[2J', b'\x1b[3J'):
                    assert sequence not in term.output, sequence
        finally:
            term.cleanup(output / 'inline.ansi')
    return {'prior_context': 'passed', 'compact_rows_inline_edit_insert_append': 'passed',
            'cancel_save_undo_unicode': 'passed', 'resize_and_tool_shrink': 'passed', 'normal_buffer_shutdown': 'passed'}


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    print(json.dumps(check(args.binary.resolve(), args.output.resolve()), indent=2))
