use crate::config;
use chrono::{DateTime, Local, Utc};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::{
    fs,
    io::Write,
    path::{Path, PathBuf},
};
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct RecentFile {
    pub path: PathBuf,
    pub last_accessed: DateTime<Utc>,
    pub access_count: u64,
    pub last_cursor_pos: usize,
    pub content_hash: String,
    pub last_modified: DateTime<Utc>,
}
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct Recent {
    pub files: Vec<RecentFile>,
    pub max_recent: usize,
}
impl Recent {
    pub fn load(dir: &Path, limit: usize) -> Result<Self, String> {
        let mut data = match fs::read_to_string(dir.join("recent.json")) {
            Ok(s) => serde_json::from_str::<Self>(&s).map_err(|e| e.to_string())?,
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => Self {
                files: vec![],
                max_recent: 20,
            },
            Err(e) => return Err(e.to_string()),
        };
        if limit > 0 {
            data.max_recent = limit;
        }
        if data.max_recent == 0 {
            data.max_recent = 20;
        }
        data.sort();
        Ok(data)
    }
    pub fn sort(&mut self) {
        let now = Utc::now();
        let score = |f: &RecentFile| {
            let hours = (now - f.last_accessed).as_seconds_f64() / 3600.;
            let frequency = if f.access_count > 1 {
                1. + (f.access_count - 1) as f64 * 0.5
            } else {
                f.access_count as f64
            };
            frequency / (1. + hours / 24.)
        };
        self.files.sort_by(|a, b| score(b).total_cmp(&score(a)));
    }
    pub fn cursor(&self, path: &Path) -> Option<usize> {
        let path = config::resolve(path).ok()?;
        let f = self.files.iter().find(|f| f.path == path)?;
        if !f.content_hash.is_empty() && f.content_hash != hash(&path).ok()? {
            return None;
        }
        Some(f.last_cursor_pos)
    }
    pub fn save(&self, dir: &Path) -> Result<(), String> {
        fs::create_dir_all(dir).map_err(|e| e.to_string())?;
        let mut temp = tempfile::NamedTempFile::new_in(dir).map_err(|e| e.to_string())?;
        temp.write_all(
            serde_json::to_string_pretty(self)
                .map_err(|e| e.to_string())?
                .as_bytes(),
        )
        .map_err(|e| e.to_string())?;
        temp.as_file().sync_all().map_err(|e| e.to_string())?;
        temp.persist(dir.join("recent.json"))
            .map_err(|e| e.to_string())?;
        Ok(())
    }
    pub fn record(dir: &Path, limit: usize, path: &Path, cursor: usize) -> Result<(), String> {
        let path = config::resolve(path)?;
        let metadata = fs::metadata(&path).map_err(|e| e.to_string())?;
        let mut recent = Self::load(dir, limit).unwrap_or(Self {
            files: vec![],
            max_recent: limit.max(1),
        });
        let old = recent.files.iter().position(|f| f.path == path);
        let count = old.map_or(1, |i| recent.files.remove(i).access_count.saturating_add(1));
        recent.files.push(RecentFile {
            content_hash: hash(&path)?,
            path,
            last_accessed: Utc::now(),
            access_count: count,
            last_cursor_pos: cursor,
            last_modified: metadata
                .modified()
                .map(DateTime::<Utc>::from)
                .unwrap_or_else(|_| Utc::now()),
        });
        recent.files.retain(|f| f.path.exists());
        recent.sort();
        recent.files.truncate(recent.max_recent);
        recent.save(dir)
    }
    pub fn listing(&self) -> String {
        if self.files.is_empty() {
            return "No recent files\n".into();
        }
        let mut text = String::from("Recent files:\n");
        for (i, f) in self.files.iter().enumerate() {
            let display = config::home()
                .and_then(|h| {
                    f.path
                        .strip_prefix(h)
                        .ok()
                        .map(|p| format!("~/{}", p.display()))
                })
                .unwrap_or_else(|| f.path.display().to_string());
            text.push_str(&format!(
                "  {}. {} (accessed {} times, last: {})\n",
                i + 1,
                display,
                f.access_count,
                f.last_accessed
                    .with_timezone(&Local)
                    .format("%Y-%m-%d %H:%M")
            ));
        }
        text.push_str("\nUse 'tdx recent <number>' to open a file\n");
        text
    }
}
fn hash(path: &Path) -> Result<String, String> {
    fs::read(path)
        .map(|s| format!("{:x}", Sha256::digest(s)))
        .map_err(|e| e.to_string())
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn retention_cursor_and_external_change() {
        let dir = tempfile::tempdir().unwrap();
        let a = dir.path().join("a.md");
        fs::write(&a, "one").unwrap();
        Recent::record(dir.path(), 1, &a, 3).unwrap();
        assert_eq!(Recent::load(dir.path(), 1).unwrap().cursor(&a), Some(3));
        fs::write(&a, "two").unwrap();
        assert_eq!(Recent::load(dir.path(), 1).unwrap().cursor(&a), None);
        let b = dir.path().join("b.md");
        fs::write(&b, "b").unwrap();
        Recent::record(dir.path(), 1, &b, 0).unwrap();
        let r = Recent::load(dir.path(), 1).unwrap();
        assert_eq!(r.files.len(), 1);
        assert_eq!(r.files[0].path, b);
    }
}
