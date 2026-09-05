#[allow(dead_code)]
#[path = "../actions.rs"]
mod actions;
#[allow(dead_code)]
#[path = "../document.rs"]
mod document;
use serde::Deserialize;
use std::io::{self, BufRead, Write};
#[derive(Deserialize)]
struct Request {
    source: String,
    actions: Vec<actions::Action>,
}
fn run() -> Result<(), Box<dyn std::error::Error>> {
    if std::env::args().nth(1).as_deref() == Some("--hold-lock") {
        let file = std::fs::OpenOptions::new()
            .read(true)
            .write(true)
            .create(true)
            .truncate(false)
            .open(std::env::args().nth(2).ok_or("lock path required")?)?;
        file.lock()?;
        println!("locked");
        io::stdout().flush()?;
        io::stdin().read_line(&mut String::new())?;
        return Ok(());
    }

    let mut out = io::BufWriter::new(io::stdout().lock());
    for line in io::stdin().lock().lines() {
        let req: Request = serde_json::from_str(&line?)?;
        let mut doc = document::Document::parse(req.source)?;
        let snapshot = |d: &document::Document| serde_json::json!({"source":d.source,"tasks":d.tasks,"headings":d.headings});
        let mut steps = vec![snapshot(&doc)];
        for action in req.actions {
            let error = match doc.apply(&action) {
                Ok((next, _)) => {
                    doc = next;
                    None
                }
                Err(e) => Some(e),
            };
            let mut step = snapshot(&doc);
            if let Some(e) = error {
                step["error"] = e.into();
            }
            steps.push(step);
        }
        serde_json::to_writer(&mut out, &steps)?;
        writeln!(out)?;
        out.flush()?;
    }
    Ok(())
}
fn main() {
    if let Err(e) = run() {
        eprintln!("{e}");
        std::process::exit(1);
    }
}
