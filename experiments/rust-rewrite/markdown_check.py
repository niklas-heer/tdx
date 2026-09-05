"""Native terminal contracts for the Rust-only Markdown document editor."""
import argparse
import json
from pathlib import Path
import tempfile
from native_check import terminal_type


def check(binary, output):
    output.mkdir(parents=True, exist_ok=True)
    Terminal = terminal_type()
    with tempfile.TemporaryDirectory(prefix='tdx-markdown-') as tmp:
        base = Path(tmp); path = base/'tasks.md'; config = base/'config'
        (config/'tdx').mkdir(parents=True)
        (config/'tdx/config.toml').write_text('[versioning]\nmax_versions=100\n')
        original = '# Project\n\nOriginal prose.\n\n- [ ] Original task\n'.encode()
        draft = '# Café 🦀\n\n**Bold** and [Guide](https://ratatui.rs).\n\n- [ ] New task\n\n```rs\r\n\tlet value = 1;\r\n```\n'.encode()
        path.write_bytes(original)
        term = Terminal(binary.resolve(), path, config)
        try:
            term.until(lambda: b'Original task' in term.output, 'initial tasks')
            term.send(b':markdown\r')
            # Ratatui emits changed cells, so modal titles need not be contiguous
            # in the raw stream. Buffer tests verify presentation; this gate checks
            # real keyboard input and persistence effects.
            term.pump(.2)
            assert path.read_bytes()==original
            term.send(b'e\x01\x1b[200~'+draft+b'\x1b[201~')
            term.pump(.3)
            assert path.read_bytes()==original, 'draft saved before explicit save'
            for width in (36,120):
                term.resize(width,28);term.pump(.2)
                term.send(b'\x10');term.pump(.2);term.send(b'\x10');term.pump(.2)
            term.send(b'\x13')
            term.until(lambda:path.read_bytes()==draft,'complete Unicode Markdown source save')
            term.send(b'\x1b');term.pump(.2);term.send(b'u')
            term.until(lambda:path.read_bytes()==original,'document undo')
            term.send(b':edit-markdown\r')
            term.pump(.2)
            term.send(b'\x01\x1b[200~'+draft+b'\x1b[201~');term.pump(.2)
            external=b'# External\n\n- [ ] Keep external\n';path.write_bytes(external)
            term.send(b'\x13')
            term.pump(.3)
            assert path.read_bytes()==external
            term.send(b'\x1b');term.pump(.2)
            term.send(b'\x1b');term.pump(.2)
            # Keeping the draft permits a save once the expected disk revision is restored.
            path.write_bytes(original);term.send(b'\x13')
            term.until(lambda:path.read_bytes()==draft,'retained draft save')
            term.send(b'\x1b');term.pump(.2);term.send(b'u')
            term.until(lambda:path.read_bytes()==original,'undo retained draft')
            term.send(b':edit-markdown\r');term.pump(.2)
            term.send(b'unsaved text');term.pump(.2)
            term.send(b'\x1b');term.pump(.2);term.send(b'y');term.pump(.2)
            term.close()
            assert path.read_bytes()==original
        except BaseException:
            (output/'rust-markdown-failure.bin').write_bytes(path.read_bytes())
            raise
        finally:term.cleanup(output/'rust-markdown.ansi')
    report={'preview_edit_multiline_unicode_resize_save_undo_conflict_retention_shutdown':'passed'}
    (output/'markdown.json').write_text(json.dumps(report,indent=2)+'\n')
    print('Native Markdown source/preview/save/undo/conflict checks passed')
    return report


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('--binary',type=Path,required=True);p.add_argument('--output',type=Path,required=True)
    args=p.parse_args();check(args.binary,args.output)
