use pulldown_cmark::{Event, Options, Parser, Tag, TagEnd};
use serde::Serialize;
use std::{
    env, fs,
    hint::black_box,
    io::{self, Write},
    process::ExitCode,
    time::Instant,
};

#[derive(Serialize)]
struct Marker {
    checked: bool,
    depth: usize,
    #[serde(skip)]
    offset: usize,
}

// Only strip a leading YAML frontmatter block; this spike does not interpret settings.
fn body_offset(source: &str) -> usize {
    let mut lines = source.split_inclusive('\n');
    let Some(first) = lines.next() else { return 0 };
    if first.trim_end() != "---" {
        return 0;
    }
    let mut offset = first.len();
    for line in lines {
        offset += line.len();
        if line.trim_end() == "---" {
            return offset;
        }
    }
    0
}

fn scan(source: &str) -> Vec<Marker> {
    let base = body_offset(source);
    let options =
        Options::ENABLE_TASKLISTS | Options::ENABLE_TABLES | Options::ENABLE_STRIKETHROUGH;
    let mut depth: usize = 0;
    let mut markers = Vec::new();
    for (event, range) in Parser::new_ext(&source[base..], options).into_offset_iter() {
        match event {
            Event::Start(Tag::List(_)) => depth += 1,
            Event::End(TagEnd::List(_)) => depth = depth.saturating_sub(1),
            Event::TaskListMarker(checked) => markers.push(Marker {
                checked,
                depth: depth.saturating_sub(1),
                offset: base + range.start,
            }),
            _ => {}
        }
    }
    markers
}

// Emit a patched copy. Never overwrite the input: this has no locking/history/fsync layer.
fn patch(source: &str, index: usize) -> Result<String, String> {
    let markers = scan(source);
    let marker = markers.get(index).ok_or("invalid task index")?;
    let start = marker.offset;
    let bytes = source.as_bytes();
    if bytes.get(start) != Some(&b'[') || bytes.get(start + 2) != Some(&b']') {
        return Err("parser marker range is not a checkbox".into());
    }
    let mut output = source.to_owned();
    output.replace_range(start + 1..start + 2, if marker.checked { " " } else { "x" });
    Ok(output)
}

fn run() -> Result<(), String> {
    let args: Vec<String> = env::args().collect();
    if args.len() != 4 {
        return Err(
            "usage: rust-probe inspect|patch|scan|patch-bench FILE INDEX_OR_ITERATIONS".into(),
        );
    }
    let source = fs::read_to_string(&args[2]).map_err(|e| e.to_string())?;
    let n: usize = args[3]
        .parse()
        .map_err(|_| "expected a non-negative integer")?;
    match args[1].as_str() {
        "inspect" => println!(
            "{}",
            serde_json::to_string(&scan(&source)).map_err(|e| e.to_string())?
        ),
        "patch" => io::stdout()
            .write_all(patch(&source, n)?.as_bytes())
            .map_err(|e| e.to_string())?,
        "scan" | "patch-bench" => {
            if n == 0 {
                return Err("iterations must be positive".into());
            }
            if let Some(path) = env::var_os("TDX_PROBE_PROFILE_READY") {
                fs::write(path, b"ready").map_err(|e| e.to_string())?;
            }
            let started = Instant::now();
            let mut checksum: usize = 0;
            for _ in 0..n {
                checksum = checksum.wrapping_add(if args[1] == "scan" {
                    black_box(scan(black_box(&source))).len()
                } else {
                    black_box(patch(black_box(&source), 0)?).len()
                });
            }
            println!(
                "{}",
                serde_json::json!({"ns_per_op":started.elapsed().as_nanos() as f64 / n as f64,"checksum":checksum})
            );
        }
        _ => return Err("unknown operation".into()),
    }
    Ok(())
}

fn main() -> ExitCode {
    match run() {
        Ok(()) => ExitCode::SUCCESS,
        Err(error) => {
            eprintln!("rust-probe: {error}");
            ExitCode::FAILURE
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn preserves_every_byte_except_the_marker() {
        let source =
            "---\r\nread-only: true\r\n---\r\n# Käse\r\n\r\n- [ ] [Link](https://example.org)\r\n";
        let output = patch(source, 0).unwrap();
        assert_eq!(output, source.replacen("[ ]", "[x]", 1));
        assert_eq!(patch(&output, 0).unwrap(), source);
    }
    #[test]
    fn ignores_fences_and_tracks_nested_tasks() {
        let source = "```md\n- [ ] fake\n```\n\n- [x] Parent\n  - [ ] Child\n";
        let markers = scan(source);
        assert_eq!(markers.len(), 2);
        assert!(markers[0].checked);
        assert_eq!(markers[1].depth, 1);
        assert!(patch(source, 2).is_err());
    }
    #[test]
    fn shared_correctness_corpus() {
        let fixtures: serde_json::Value =
            serde_json::from_str(include_str!("../../fixtures.json")).unwrap();
        for fixture in fixtures.as_array().unwrap() {
            let source = fixture["source"].as_str().unwrap();
            let markers = scan(source);
            assert_eq!(
                serde_json::to_value(&markers).unwrap(),
                fixture["markers"],
                "{}",
                fixture["name"]
            );
            for index in 0..markers.len() {
                let output = patch(source, index).unwrap();
                assert_eq!(output.len(), source.len());
                assert_eq!(
                    output
                        .bytes()
                        .zip(source.bytes())
                        .filter(|(a, b)| a != b)
                        .count(),
                    1
                );
                let mut expected = fixture["markers"].clone();
                expected[index]["checked"] = (!markers[index].checked).into();
                assert_eq!(
                    serde_json::to_value(scan(&output)).unwrap(),
                    expected,
                    "{} task {}",
                    fixture["name"],
                    index
                );
            }
            assert!(patch(source, markers.len()).is_err());
        }
    }
}
