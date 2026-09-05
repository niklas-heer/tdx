use std::{
    io::Write,
    process::{Command, Stdio},
};
fn commands(copy: bool) -> Vec<Vec<&'static str>> {
    if cfg!(target_os = "macos") {
        vec![vec![if copy { "pbcopy" } else { "pbpaste" }]]
    } else if cfg!(windows) {
        vec![if copy {
            vec![
                "powershell.exe",
                "-NoProfile",
                "-NonInteractive",
                "-Command",
                "$text = [Console]::In.ReadToEnd(); Set-Clipboard -Value $text",
            ]
        } else {
            vec![
                "powershell.exe",
                "-NoProfile",
                "-NonInteractive",
                "-Command",
                "Get-Clipboard -Raw",
            ]
        }]
    } else if copy {
        vec![
            vec!["wl-copy"],
            vec!["xclip", "-selection", "clipboard"],
            vec!["xsel", "--clipboard", "--input"],
        ]
    } else {
        vec![
            vec!["wl-paste", "--no-newline"],
            vec!["xclip", "-selection", "clipboard", "-o"],
            vec!["xsel", "--clipboard", "--output"],
        ]
    }
}
pub fn copy(text: &str) -> Result<(), String> {
    for args in commands(true) {
        if let Ok(mut child) = Command::new(args[0])
            .args(&args[1..])
            .stdin(Stdio::piped())
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .spawn()
        {
            let wrote = child
                .stdin
                .take()
                .is_some_and(|mut input| input.write_all(text.as_bytes()).is_ok());
            if child.wait().is_ok_and(|s| s.success()) && wrote {
                return Ok(());
            }
        }
    }
    Err("clipboard unavailable".into())
}
pub fn paste() -> Result<String, String> {
    for args in commands(false) {
        if let Ok(out) = Command::new(args[0]).args(&args[1..]).output()
            && out.status.success()
        {
            return String::from_utf8(out.stdout).map_err(|e| e.to_string());
        }
    }
    Err("clipboard unavailable".into())
}
