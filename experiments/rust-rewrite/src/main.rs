mod actions;
mod clipboard;
mod config;
#[cfg(any(windows, test))]
mod console_input;
mod document;
mod editor;
mod history;
mod input;
mod presentation;
mod recent;
mod store;
mod tui;

use config::{Config, Overrides, Settings};
use editor::Editor;
use recent::Recent;
use std::{
    env,
    io::{self, IsTerminal, Write},
    path::PathBuf,
    process::ExitCode,
};
use store::Store;
const HELP: &str = "tdx - Markdown todo manager (Rust implementation)\n\nUsage: tdx [file.md] [command] [args]\n       tdx --file PATH [command] [args]\nOptions: -f/--file PATH, -r/--read-only, --show-headings, -m/--max-visible N\nList options: --json, --status all|open|done, --tag TAG (repeatable)\nCommands: list, add TEXT, edit INDEX TEXT, toggle INDEX, delete INDEX, last, recent [NUMBER|clear], help\nHistory: versions [--json], show-version ID, restore ID\nUse -- before literal task text beginning with '-'.\n\nTUI: j/k or arrows, gg/G, counted navigation; space/Enter toggle; n/N new; e edit; d delete; c copy; m move; Tab/Shift-Tab indent/outdent; u undo; / search; t/p/D filters; s/S sections; r recent files; : command palette; ? help; Esc quit.\nInput: Unicode typing/paste, arrows, Home/End, Backspace/Delete, Ctrl-A/E, Ctrl-Y paste.\n";
#[derive(Default, Debug)]
struct Args {
    file: PathBuf,
    command: String,
    values: Vec<String>,
    flags: Overrides,
    json: bool,
    status: String,
    tags: Vec<String>,
}
#[expect(
    clippy::too_many_lines,
    reason = "CLI dispatch mirrors the Go command contract in one place"
)]
#[expect(
    clippy::case_sensitive_file_extension_comparisons,
    reason = "Match Go positional .md argument detection; --file accepts any extension"
)]
fn parse_args(args: Vec<String>, config: &Config) -> Result<Args, String> {
    let mut parsed = Args {
        file: config.defaults.file.clone().into(),
        ..Args::default()
    };
    let mut args = args.into_iter();
    let mut literal = false;
    let mut file_set = false;
    let mut query_flags = false;
    while let Some(arg) = args.next() {
        if !literal && arg == "--" {
            literal = true;
            continue;
        }
        if !literal && arg.starts_with('-') {
            let (name, inline) = arg
                .split_once('=')
                .map_or((arg.as_str(), None), |(n, v)| (n, Some(v.to_owned())));
            match name {
                "--file" | "-f" | "--status" | "--tag" | "--max-visible" | "-m" => {
                    let value = inline
                        .or_else(|| args.next())
                        .ok_or_else(|| format!("{name} requires a value"))?;
                    match name {
                        "--file" | "-f" => {
                            if value.is_empty() || file_set {
                                return Err("specify one non-empty file path".into());
                            }
                            parsed.file = value.into();
                            file_set = true;
                        }
                        "--status" => {
                            if !["all", "open", "done"].contains(&value.as_str()) {
                                return Err("status must be all, open, or done".into());
                            }
                            parsed.status = value;
                            query_flags = true;
                        }
                        "--tag" => {
                            let value = value.strip_prefix('#').unwrap_or(&value);
                            if value.is_empty() || value.chars().any(char::is_whitespace) {
                                return Err("invalid tag".into());
                            }
                            parsed.tags.push(value.into());
                            query_flags = true;
                        }
                        _ => {
                            let n = value
                                .parse::<isize>()
                                .map_err(|_| "max-visible requires a non-negative integer")?;
                            if n < 0 {
                                return Err("max-visible requires a non-negative integer".into());
                            }
                            parsed.flags.max_visible = Some(n);
                        }
                    }
                }
                "--json" | "--read-only" | "-r" | "--show-headings" | "--help" | "-h"
                | "--version" | "-v" | "--debug-config" => {
                    if inline.is_some() {
                        return Err(format!("{name} does not take a value"));
                    }
                    match name {
                        "--json" => {
                            parsed.json = true;
                            query_flags = true;
                        }
                        "--read-only" | "-r" => parsed.flags.read_only = true,
                        "--show-headings" => parsed.flags.show_headings = true,
                        "--help" | "-h" => {
                            parsed.command = "help".into();
                            parsed.values.clear();
                            return Ok(parsed);
                        }
                        _ => {
                            if !parsed.command.is_empty() {
                                return Err(format!("use {name} on its own"));
                            }
                            parsed.command = if name == "--debug-config" {
                                "debug-config"
                            } else {
                                "version"
                            }
                            .into();
                        }
                    }
                }
                _ => {
                    return Err(format!(
                        "unknown option {name}; use -- before literal task text"
                    ));
                }
            }
        } else if parsed.command.is_empty() {
            if !file_set && arg.ends_with(".md") {
                parsed.file = arg.into();
                file_set = true;
            } else {
                parsed.command = arg;
            }
        } else {
            parsed.values.push(arg);
        }
    }
    if query_flags
        && parsed.command != "list"
        && !(parsed.command == "versions"
            && parsed.json
            && parsed.status.is_empty()
            && parsed.tags.is_empty())
    {
        return Err("query flags require list".into());
    }
    let valid = match parsed.command.as_str() {
        "" | "list" | "help" | "version" | "debug-config" | "versions" | "last" => {
            parsed.values.is_empty()
        }
        "add" => !parsed.values.is_empty(),
        "edit" => parsed.values.len() >= 2,
        "toggle" | "delete" | "show-version" | "restore" => parsed.values.len() == 1,
        "recent" => parsed.values.len() <= 1,
        _ => return Err(format!("unknown command {}", parsed.command)),
    };
    if !valid {
        return Err("wrong argument count; use --help".into());
    }
    Ok(parsed)
}
fn version() -> String {
    include_str!("../../../tdx.toml")
        .lines()
        .find_map(|l| {
            l.strip_prefix("version = ")
                .map(|s| s.trim_matches('"').to_owned())
        })
        .unwrap_or_else(|| env!("CARGO_PKG_VERSION").into())
}
#[expect(
    clippy::too_many_lines,
    reason = "CLI dispatch mirrors the Go command contract in one place"
)]
fn run() -> Result<(), String> {
    let config = Config::load();
    let mut args = parse_args(env::args().skip(1).collect(), &config)?;
    match args.command.as_str() {
        "help" => {
            print!("{HELP}");
            return Ok(());
        }
        "version" => {
            println!("tdx v{}", version());
            return Ok(());
        }
        "debug-config" => {
            let themes = config::themes();
            let colors = Some(themes.get(&config.theme.name).unwrap_or(&config.colors));
            println!(
                "Theme: {}\nColors.Accent: {}\nColors.Success: {}\nDisplay.CheckSymbol: {}\nDisplay.SelectMarker: {}\nDefaults.File: {}\nDefaults.MaxVisible: {}\nDefaults.WordWrap: {}\nDefaults.ShowHeadings: {}\nDefaults.ReadOnly: {}\nDefaults.FilterDone: {}\nRecent.MaxFiles: {}",
                config.theme.name,
                colors
                    .and_then(|c| c.get("Accent"))
                    .map_or("", String::as_str),
                colors
                    .and_then(|c| c.get("Success"))
                    .map_or("", String::as_str),
                config.display.check_symbol,
                config.display.select_marker,
                config.defaults.file,
                config.defaults.max_visible,
                config.defaults.word_wrap,
                config.defaults.show_headings,
                config.defaults.read_only,
                config.defaults.filter_done,
                config.recent.max_files
            );
            return Ok(());
        }
        "recent" | "last" => {
            let dir = config::directory()?;
            if args.values.first().is_some_and(|v| v == "clear") {
                Recent {
                    files: vec![],
                    max_recent: config.recent.max_files,
                }
                .save(&dir)?;
                println!("Recent files cleared");
                return Ok(());
            }
            let recent = Recent::load(&dir, config.recent.max_files)?;
            if args.command == "recent" && args.values.is_empty() {
                print!("{}", recent.listing());
                return Ok(());
            }
            let index = if args.command == "last" {
                0
            } else {
                positive(&args.values[0])?
            };
            args.file.clone_from(
                &recent
                    .files
                    .get(index)
                    .ok_or("no matching recent file")?
                    .path,
            );
            args.command.clear();
            args.values.clear();
        }
        _ => {}
    }
    let mutation = ["add", "edit", "toggle", "delete", "restore"].contains(&args.command.as_str());
    let index = if ["edit", "toggle", "delete"].contains(&args.command.as_str()) {
        positive(&args.values[0])?
    } else {
        0
    };
    let version_id = if ["show-version", "restore"].contains(&args.command.as_str()) {
        let v = args.values[0]
            .parse::<i64>()
            .map_err(|_| "invalid version ID")?;
        if v <= 0 {
            return Err("invalid version ID".into());
        }
        v
    } else {
        0
    };
    args.file = config::resolve(&args.file)?;
    let mut editor = Editor::new(Store::load(&args.file)?, false)?;
    let settings = Settings::new(&config, &editor.doc.metadata, &args.flags);
    if mutation && settings.read_only {
        return Err("read-only mode: task editing is disabled".into());
    }
    let interactive = args.command.is_empty();
    editor.readonly = settings.read_only;
    if args.command != "list" {
        editor
            .store
            .enable_history_with_limit(mutation || interactive, config.versioning.max_versions)?;
    }
    let result = (|| match args.command.as_str() {
        "" => {
            if !io::stdin().is_terminal() || !io::stdout().is_terminal() {
                return tui::piped(&mut editor, &args.file, config.clone(), args.flags.clone())
                    .map_err(|e| e.to_string());
            }
            tui::run(&mut editor, &args.file, config.clone(), args.flags.clone())
                .map_err(|e| e.to_string())
        }
        "list" => {
            let rows: Vec<_> = editor
                .doc
                .query()
                .iter()
                .filter(|t| {
                    (args.status != "open" || !t.checked)
                        && (args.status != "done" || t.checked)
                        && args.tags.iter().all(|tag| t.tags.contains(tag))
                })
                .collect();
            let mut out = io::BufWriter::new(io::stdout().lock());
            if args.json {
                serde_json::to_writer_pretty(&mut out, &rows).map_err(|e| e.to_string())?;
                writeln!(out).map_err(|e| e.to_string())?;
            } else if rows.is_empty() {
                writeln!(out, "No todos found").map_err(|e| e.to_string())?;
            } else {
                for t in rows {
                    writeln!(
                        out,
                        "  {}. [{}] {}",
                        t.index,
                        if t.checked {
                            &config.display.check_symbol
                        } else {
                            " "
                        },
                        t.text
                    )
                    .map_err(|e| e.to_string())?;
                }
            }
            out.flush().map_err(|e| e.to_string())
        }
        "versions" => {
            let versions = editor.store.versions()?;
            if args.json {
                println!(
                    "{}",
                    serde_json::to_string_pretty(&versions).map_err(|e| e.to_string())?
                );
            } else {
                for v in versions {
                    println!("{} {}", v.id, v.created_at);
                }
            }
            Ok(())
        }
        "show-version" => io::stdout()
            .write_all(editor.store.version(version_id)?.as_bytes())
            .map_err(|e| e.to_string()),
        "restore" => editor.restore(version_id),
        op => {
            let prior = editor.doc.tasks.get(index).cloned();
            let text = if op == "add" {
                args.values.join(" ")
            } else if op == "edit" {
                args.values[1..].join(" ")
            } else {
                String::new()
            };
            editor.apply(op, index, &text)?;
            match op {
                "add" => println!("✓ Added: {text}"),
                "edit" => println!("✓ Edited: {text}"),
                "delete" => println!("✓ Deleted: {}", prior.ok_or("task no longer exists")?.text),
                "toggle" => {
                    let t = prior.ok_or("task no longer exists")?;
                    println!(
                        "✓ Toggled: [{}] {}",
                        if t.checked {
                            " "
                        } else {
                            &config.display.check_symbol
                        },
                        t.text
                    );
                }
                _ => {}
            }
            Ok(())
        }
    })();
    let finish = editor.store.finish();
    match (result, finish) {
        (Err(a), Err(b)) => Err(format!("{a}; {b}")),
        (Err(a), _) => Err(a),
        (_, Err(b)) => Err(b),
        _ => Ok(()),
    }
}
fn positive(value: &str) -> Result<usize, String> {
    value
        .parse::<usize>()
        .ok()
        .filter(|i| *i > 0)
        .map(|i| i - 1)
        .ok_or_else(|| "invalid index: use a positive integer".into())
}
fn main() -> ExitCode {
    match run() {
        Ok(()) => ExitCode::SUCCESS,
        Err(e) => {
            eprintln!("tdx: {e}");
            ExitCode::FAILURE
        }
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn argument_validation_and_literal_flags() {
        let c = Config::default();
        assert_eq!(
            parse_args(vec!["add".into(), "--".into(), "--read-only".into()], &c)
                .unwrap()
                .values,
            ["--read-only"]
        );
        assert!(parse_args(vec!["--unknown".into()], &c).is_err());
        assert!(parse_args(vec!["list".into(), "--status".into(), "bad".into()], &c).is_err());
    }
}
