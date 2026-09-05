mod document;
mod editor;
mod history;
mod store;
mod tui;

use editor::Editor;
use std::{
    env,
    io::{self, IsTerminal, Write},
    path::PathBuf,
    process::ExitCode,
};
use store::Store;

const HELP: &str = "tdx-rust — experimental Rust CLI + TUI with version history\n\nUsage: tdx-rust [--file PATH] [--read-only] [COMMAND]\n  list [--json] [--status all|open|done] [--tag TAG]\n  add TEXT | edit INDEX TEXT | toggle INDEX | delete INDEX\n  versions [--json] | show-version ID | restore ID\n  No command: interactive editor. Use -- before literal text starting with '-'.\n\nTUI: j/k or arrows, space toggle, a add, e edit, d delete, u undo, r reload, v versions, q quit.\nCommands: :versions, :reload, :force-save. History: Enter then y restores; Esc cancels.\nInput: Enter save, Esc cancel, Ctrl-U clear, Unicode paste; cursor at end.\nSingle-line parent labels can be edited; deleting parents and multiline task bodies remain unsupported.\nHistory uses the Go-compatible versions.sqlite in the tdx config directory.\n";

#[derive(Default, Debug)]
struct Args {
    file: PathBuf,
    command: String,
    values: Vec<String>,
    readonly: bool,
    json: bool,
    status: String,
    tags: Vec<String>,
}
fn parse_args(args: Vec<String>) -> Result<Args, String> {
    let mut parsed = Args {
        file: "todo.md".into(),
        ..Args::default()
    };
    let mut args = args.into_iter();
    let mut literal = false;
    let mut file_set = false;
    let mut list_flags = false;
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
                "--file" | "-f" | "--status" | "--tag" => {
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
                                return Err("invalid status".into());
                            }
                            parsed.status = value;
                            list_flags = true;
                        }
                        _ => {
                            let tag = value.trim_start_matches('#');
                            if tag.is_empty() || tag.chars().any(char::is_whitespace) {
                                return Err("invalid tag".into());
                            }
                            parsed.tags.push(tag.to_owned());
                            list_flags = true;
                        }
                    }
                }
                "--json" | "--read-only" | "-r" | "--help" | "-h" | "--version" | "-v" => {
                    if inline.is_some() {
                        return Err(format!("{name} does not take a value"));
                    }
                    match name {
                        "--json" => {
                            parsed.json = true;
                            list_flags = true;
                        }
                        "--read-only" | "-r" => parsed.readonly = true,
                        "--help" | "-h" => {
                            parsed.command = "help".into();
                            return Ok(parsed);
                        }
                        _ => parsed.command = "version".into(),
                    }
                }
                _ => return Err(format!("unsupported option {name}")),
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
    if list_flags
        && parsed.command != "list"
        && !(parsed.command == "versions"
            && parsed.json
            && parsed.status.is_empty()
            && parsed.tags.is_empty())
    {
        return Err("query flags require list".into());
    }
    let valid_count = match parsed.command.as_str() {
        "" | "list" | "help" | "version" | "versions" => parsed.values.is_empty(),
        "add" => parsed.values.len() == 1,
        "edit" => parsed.values.len() == 2,
        "toggle" | "delete" | "show-version" | "restore" => parsed.values.len() == 1,
        _ => return Err(format!("unsupported command {}", parsed.command)),
    };
    if !valid_count {
        return Err("wrong argument count; use --help".into());
    }
    Ok(parsed)
}
fn run() -> Result<(), String> {
    let args = parse_args(env::args().skip(1).collect())?;
    match args.command.as_str() {
        "help" => {
            print!("{HELP}");
            return Ok(());
        }
        "version" => {
            println!("tdx-rust prototype {}", env!("CARGO_PKG_VERSION"));
            return Ok(());
        }
        _ => {}
    }
    let mutation = ["add", "edit", "toggle", "delete", "restore"].contains(&args.command.as_str());
    if args.readonly && mutation {
        return Err("read-only: editing is disabled".into());
    }
    if args.command.is_empty() && (!io::stdin().is_terminal() || !io::stdout().is_terminal()) {
        return Err("interactive editor requires a terminal; use list --json for scripting".into());
    }
    let index = if ["edit", "toggle", "delete"].contains(&args.command.as_str()) {
        args.values[0]
            .parse::<usize>()
            .ok()
            .and_then(|i| i.checked_sub(1))
            .ok_or("index must be a positive integer")?
    } else {
        0
    };
    let version_id = if ["show-version", "restore"].contains(&args.command.as_str()) {
        args.values[0]
            .parse::<i64>()
            .ok()
            .filter(|i| *i > 0)
            .ok_or("version ID must be a positive integer")?
    } else {
        0
    };
    let mut editor = Editor::new(Store::load(&args.file)?, args.readonly)?;
    if args.command.is_empty() {
        editor.doc.query()?;
    }
    if mutation && editor.doc.readonly {
        return Err("read-only frontmatter: editing is disabled".into());
    }
    if args.command != "list" {
        editor
            .store
            .enable_history(mutation || args.command.is_empty())?;
    }
    let result = (|| {
        match args.command.as_str() {
            "" => tui::run(&mut editor, &args.file).map_err(|e| e.to_string())?,
            "versions" => {
                let versions = editor.store.versions()?;
                let mut out = io::BufWriter::new(io::stdout().lock());
                if args.json {
                    serde_json::to_writer_pretty(&mut out, &versions).map_err(|e| e.to_string())?;
                    writeln!(out).map_err(|e| e.to_string())?;
                } else {
                    for version in versions {
                        writeln!(out, "{}  {}", version.id, version.created_at)
                            .map_err(|e| e.to_string())?;
                    }
                }
                out.flush().map_err(|e| e.to_string())?;
            }
            "show-version" => {
                io::stdout()
                    .write_all(editor.store.version(version_id)?.as_bytes())
                    .map_err(|e| e.to_string())?;
            }
            "restore" => editor.restore(version_id)?,
            "list" => {
                let tasks: Vec<_> = editor
                    .doc
                    .query()?
                    .iter()
                    .filter(|t| {
                        !(args.status == "open" && t.checked || args.status == "done" && !t.checked)
                            && args.tags.iter().all(|tag| t.tags.contains(tag))
                    })
                    .collect();
                let mut out = io::BufWriter::new(io::stdout().lock());
                if args.json {
                    serde_json::to_writer_pretty(&mut out, &tasks).map_err(|e| e.to_string())?;
                    writeln!(out).map_err(|e| e.to_string())?;
                } else if tasks.is_empty() {
                    writeln!(out, "No todos found").map_err(|e| e.to_string())?;
                } else {
                    for task in tasks {
                        writeln!(
                            out,
                            "  {}. [{}] {}",
                            task.index,
                            if task.checked { "x" } else { " " },
                            task.text
                        )
                        .map_err(|e| e.to_string())?;
                    }
                }
                out.flush().map_err(|e| e.to_string())?;
            }
            op => {
                let text = if op == "edit" {
                    &args.values[1]
                } else if op == "add" {
                    &args.values[0]
                } else {
                    ""
                };
                editor.apply(op, index, text)?;
            }
        }
        Ok(())
    })();
    match (result, editor.store.finish()) {
        (Ok(()), Ok(())) => Ok(()),
        (Err(e), Ok(())) | (Ok(()), Err(e)) => Err(e),
        (Err(e), Err(close)) => Err(format!("{e}; {close}")),
    }
}
fn main() -> ExitCode {
    match run() {
        Ok(()) => ExitCode::SUCCESS,
        Err(error) => {
            eprintln!("tdx-rust: {error}");
            ExitCode::FAILURE
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn argument_validation_and_literal_flags() {
        let parse = |args: &[&str]| parse_args(args.iter().map(|s| s.to_string()).collect());
        assert!(parse(&["toggle", "1", "extra"]).is_err());
        assert!(parse(&["--json", "add", "one"]).is_err());
        assert!(parse(&["--file", "a", "--file", "b"]).is_err());
        assert_eq!(
            parse(&["add", "--", "--json"]).unwrap().values,
            vec!["--json"]
        );
    }
}
