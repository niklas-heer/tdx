"""Render exported, real Ratatui cell buffers as portable SVG previews."""
import argparse
import html
import json
from pathlib import Path
import re


def color(value, fallback):
    if value == 'Reset': return fallback
    if value.startswith('Rgb('):
        return '#%02x%02x%02x' % tuple(map(int, re.findall(r'\d+', value)))
    return {'Black':'#15161e', 'White':'#ffffff','Gray':'#c0caf5','DarkGray':'#565f89','Red':'#f7768e','Green':'#c3e88d','Blue':'#7aa2f7','Cyan':'#7dcfff','Magenta':'#bb9af7','Yellow':'#e0af68'}.get(value, fallback)


def main():
    p = argparse.ArgumentParser(); p.add_argument('directory', type=Path); args = p.parse_args()
    screens = json.loads((args.directory/'ratatui-screens.json').read_text())
    cards = []
    for screen in screens:
        w, h = screen['width'], screen['height']; cellw, cellh, pad = 9, 20, 20
        svg = [f'<svg xmlns="http://www.w3.org/2000/svg" width="{w*cellw+pad*2}" height="{h*cellh+pad*2}" viewBox="0 0 {w*cellw+pad*2} {h*cellh+pad*2}">', '<rect width="100%" height="100%" rx="10" fill="#1a1b26"/>', '<g font-family="Menlo,DejaVu Sans Mono,monospace" font-size="14">']
        # Draw backgrounds first so double-width glyphs are never painted over.
        for index,c in enumerate(screen['cells']):
            fg,bg = color(c['fg'],'#c0caf5'),color(c['bg'],'#1a1b26')
            if 'REVERSED' in c['modifiers']: fg,bg = bg,fg
            x,y = pad+(index%w)*cellw,pad+(index//w)*cellh
            if bg != '#1a1b26': svg.append(f'<rect x="{x}" y="{y}" width="{cellw}" height="{cellh}" fill="{bg}"/>')
        for index,c in enumerate(screen['cells']):
            fg,bg = color(c['fg'],'#c0caf5'),color(c['bg'],'#1a1b26')
            if 'REVERSED' in c['modifiers']: fg,bg = bg,fg
            x,y = pad+(index%w)*cellw,pad+(index//w)*cellh
            weight = 'bold' if 'BOLD' in c['modifiers'] else 'normal'
            decoration = 'underline' if 'UNDERLINED' in c['modifiers'] else 'none'
            svg.append(f'<text x="{x}" y="{y+15}" fill="{fg}" font-weight="{weight}" text-decoration="{decoration}" xml:space="preserve">{html.escape(c["text"])}</text>')
        svg.extend(['</g>','</svg>'])
        (args.directory/f'{screen["name"]}.svg').write_text('\n'.join(svg))
        cards.append(f'<section><h2>{html.escape(screen["name"].title())}</h2><img src="{screen["name"]}.svg" alt="Actual Ratatui {screen["name"]} screen"/></section>')
    (args.directory/'index.html').write_text('<!doctype html><meta charset="utf-8"><title>tdx · Ratatui</title><style>body{background:#11131b;color:#c0caf5;font:16px system-ui;margin:32px}h1{color:#7aa2f7}h2{font-size:16px;font-weight:500}section{margin:28px 0}img{max-width:100%;height:auto}p{color:#999fb7}</style><h1>tdx · Ratatui</h1><p>Actual Rust renderer output with sample Markdown. Terminal palette: Tokyo Night.</p>'+''.join(cards))
    print(args.directory/'index.html')

if __name__ == '__main__': main()
