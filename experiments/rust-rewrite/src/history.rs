use rusqlite::{Connection, OptionalExtension, params};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::{
    collections::HashMap,
    fs,
    io::Read,
    path::{Path, PathBuf},
    time::Duration,
};

const SCHEMA: &str = "
CREATE TABLE IF NOT EXISTS files (
 id INTEGER PRIMARY KEY AUTOINCREMENT, file_path TEXT NOT NULL UNIQUE);
CREATE TABLE IF NOT EXISTS file_versions (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
 version_hash TEXT NOT NULL, content BLOB NOT NULL,
 created_at DATETIME DEFAULT CURRENT_TIMESTAMP, commit_message TEXT,
 UNIQUE(file_id, version_hash));
CREATE INDEX IF NOT EXISTS idx_file_versions_file_id ON file_versions(file_id);
";
// Bound decompression of corrupted/untrusted history. Ordinary document input
// remains unrestricted; reads of snapshots above this prototype limit fail.
const MAX_SNAPSHOT_BYTES: u64 = 64 * 1024 * 1024;

#[derive(Clone, Debug, Serialize)]
pub struct Version {
    pub id: i64,
    pub created_at: String,
}
#[derive(Deserialize)]
struct Config {
    #[serde(default)]
    versioning: Versioning,
}
#[derive(Deserialize)]
#[serde(default)]
struct Versioning {
    max_versions: i64,
}
impl Default for Versioning {
    fn default() -> Self {
        Self { max_versions: 100 }
    }
}

pub fn max_versions(config_dir: &Path) -> Result<i64, String> {
    match fs::read_to_string(config_dir.join("config.toml")) {
        Ok(text) => toml::from_str::<Config>(&text)
            .map(|c| c.versioning.max_versions)
            .map_err(|e| format!("invalid versioning config: {e}")),
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(100),
        Err(e) => Err(format!("read versioning config: {e}")),
    }
}

pub struct History {
    conn: Connection,
    ids: HashMap<PathBuf, i64>,
    encoder: zstd::bulk::Compressor<'static>,
    max_versions: i64,
}
impl History {
    pub fn open(directory: &Path, max_versions: i64) -> Result<Self, String> {
        fs::create_dir_all(directory).map_err(|e| e.to_string())?;
        let conn =
            Connection::open(directory.join("versions.sqlite")).map_err(|e| e.to_string())?;
        conn.busy_timeout(Duration::from_secs(5))
            .map_err(|e| e.to_string())?;
        conn.execute_batch("PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;")
            .map_err(|e| e.to_string())?;
        conn.execute_batch(SCHEMA).map_err(|e| e.to_string())?;
        Ok(Self {
            conn,
            ids: HashMap::new(),
            encoder: zstd::bulk::Compressor::new(1).map_err(|e| e.to_string())?,
            max_versions,
        })
    }
    fn id(&mut self, path: &Path) -> Result<i64, String> {
        if let Some(id) = self.ids.get(path) {
            return Ok(*id);
        }
        let text = path.to_str().ok_or("history paths must be UTF-8")?;
        self.conn
            .execute("INSERT OR IGNORE INTO files (file_path) VALUES (?)", [text])
            .map_err(|e| e.to_string())?;
        let id = self
            .conn
            .query_row("SELECT id FROM files WHERE file_path = ?", [text], |row| {
                row.get(0)
            })
            .map_err(|e| e.to_string())?;
        self.ids.insert(path.to_owned(), id);
        Ok(id)
    }
    pub fn capture(&mut self, path: &Path, content: &str) -> Result<(), String> {
        let id = self.id(path)?;
        let hash = format!("{:x}", Sha256::digest(content.as_bytes()));
        let compressed = self
            .encoder
            .compress(content.as_bytes())
            .map_err(|e| e.to_string())?;
        self.conn.execute("INSERT OR IGNORE INTO file_versions (file_id, version_hash, content) VALUES (?, ?, ?)", params![id, hash, compressed]).map_err(|e| e.to_string())?;
        self.prune(id)
    }
    fn prune(&self, id: i64) -> Result<(), String> {
        if self.max_versions <= 0 {
            return Ok(());
        }
        self.conn.execute("DELETE FROM file_versions WHERE file_id = ? AND id NOT IN (SELECT id FROM file_versions WHERE file_id = ? ORDER BY id DESC LIMIT ?)", params![id, id, self.max_versions]).map_err(|e| e.to_string())?;
        Ok(())
    }
    pub fn list(&mut self, path: &Path) -> Result<Vec<Version>, String> {
        let id = self.id(path)?;
        let mut statement = self
            .conn
            .prepare("SELECT id, created_at FROM file_versions WHERE file_id = ? ORDER BY id DESC")
            .map_err(|e| e.to_string())?;
        statement
            .query_map([id], |row| {
                Ok(Version {
                    id: row.get(0)?,
                    created_at: row.get(1)?,
                })
            })
            .map_err(|e| e.to_string())?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| e.to_string())
    }
    pub fn read(&mut self, path: &Path, version: i64) -> Result<String, String> {
        let id = self.id(path)?;
        let (compressed, hash): (Vec<u8>, String) = self
            .conn
            .query_row(
                "SELECT content, version_hash FROM file_versions WHERE file_id = ? AND id = ?",
                params![id, version],
                |row| Ok((row.get(0)?, row.get(1)?)),
            )
            .optional()
            .map_err(|e| e.to_string())?
            .ok_or("version not found for this file")?;
        let mut output = Vec::new();
        zstd::stream::read::Decoder::new(compressed.as_slice())
            .map_err(|e| format!("corrupt snapshot: {e}"))?
            .take(MAX_SNAPSHOT_BYTES + 1)
            .read_to_end(&mut output)
            .map_err(|e| format!("corrupt snapshot: {e}"))?;
        if output.len() as u64 > MAX_SNAPSHOT_BYTES {
            return Err("snapshot exceeds prototype 64 MiB recovery limit".into());
        }
        if format!("{:x}", Sha256::digest(&output)) != hash {
            return Err("snapshot hash mismatch".into());
        }
        String::from_utf8(output).map_err(|e| format!("snapshot is not UTF-8: {e}"))
    }
    pub fn close(self) -> Result<(), String> {
        let mut errors = Vec::new();
        for id in self.ids.values() {
            if let Err(e) = self.prune(*id) {
                errors.push(format!("prune: {e}"));
            }
        }
        let checkpoint: Result<(i64, i64, i64), _> =
            self.conn
                .query_row("PRAGMA wal_checkpoint(TRUNCATE)", [], |r| {
                    Ok((r.get(0)?, r.get(1)?, r.get(2)?))
                });
        match checkpoint {
            Ok((0, _, _)) => {}
            Ok(_) => errors.push("history checkpoint remained busy".into()),
            Err(e) => errors.push(format!("checkpoint: {e}")),
        }
        if let Err((_, e)) = self.conn.close() {
            errors.push(format!("close: {e}"));
        }
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors.join("; "))
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn retention_dedup_and_per_file_identity() {
        let dir = tempfile::tempdir().unwrap();
        let mut h = History::open(dir.path(), 3).unwrap();
        let a = dir.path().join("a.md");
        let b = dir.path().join("b.md");
        for i in 0..5 {
            h.capture(&a, &format!("- [ ] café {i}\r\n")).unwrap();
        }
        h.capture(&a, "- [ ] café 4\r\n").unwrap();
        h.capture(&b, "other").unwrap();
        let versions = h.list(&a).unwrap();
        assert_eq!(versions.len(), 3);
        assert_eq!(h.list(&b).unwrap().len(), 1);
        assert_eq!(h.read(&a, versions[0].id).unwrap(), "- [ ] café 4\r\n");
        assert!(h.read(&b, versions[0].id).is_err());
        h.close().unwrap();
    }
    #[test]
    fn corrupt_hash_and_zstd_are_rejected() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("a.md");
        let mut h = History::open(dir.path(), 0).unwrap();
        h.capture(&path, "hello").unwrap();
        let id = h.list(&path).unwrap()[0].id;
        h.conn
            .execute("UPDATE file_versions SET version_hash='bad'", [])
            .unwrap();
        assert!(h.read(&path, id).unwrap_err().contains("hash mismatch"));
        h.conn
            .execute("UPDATE file_versions SET content=X'ff'", [])
            .unwrap();
        assert!(h.read(&path, id).is_err());
    }
    #[test]
    fn contention_is_bounded_and_recoverable() {
        let dir = tempfile::tempdir().unwrap();
        let mut h = History::open(dir.path(), 100).unwrap();
        let timeout: i64 = h
            .conn
            .query_row("PRAGMA busy_timeout", [], |r| r.get(0))
            .unwrap();
        assert_eq!(timeout, 5000);
        // Shorten only this test's wait; exercise the real SQLite writer lock.
        h.conn.busy_timeout(Duration::from_millis(20)).unwrap();
        let blocker = Connection::open(dir.path().join("versions.sqlite")).unwrap();
        blocker.execute_batch("BEGIN IMMEDIATE").unwrap();
        let path = dir.path().join("a.md");
        assert!(h.capture(&path, "one").unwrap_err().contains("locked"));
        blocker.execute_batch("ROLLBACK").unwrap();
        h.capture(&path, "one").unwrap();
        assert_eq!(h.list(&path).unwrap().len(), 1);
        h.close().unwrap();
    }
    #[test]
    fn shutdown_reports_busy_checkpoint() {
        let dir = tempfile::tempdir().unwrap();
        let mut h = History::open(dir.path(), 100).unwrap();
        h.conn.busy_timeout(Duration::from_millis(20)).unwrap();
        let path = dir.path().join("a.md");
        h.capture(&path, "one").unwrap();
        let reader = Connection::open(dir.path().join("versions.sqlite")).unwrap();
        reader
            .execute_batch("BEGIN; SELECT * FROM file_versions;")
            .unwrap();
        h.capture(&path, "two").unwrap();
        assert!(h.close().unwrap_err().contains("checkpoint remained busy"));
        reader.execute_batch("ROLLBACK").unwrap();
    }
    #[test]
    fn retention_config_defaults_and_validation() {
        let dir = tempfile::tempdir().unwrap();
        assert_eq!(max_versions(dir.path()).unwrap(), 100);
        fs::write(
            dir.path().join("config.toml"),
            "[theme]\nname='nord'\n[versioning]\nmax_versions=0\n",
        )
        .unwrap();
        assert_eq!(max_versions(dir.path()).unwrap(), 0);
        fs::write(
            dir.path().join("config.toml"),
            "[versioning]\nmax_versions='bad'\n",
        )
        .unwrap();
        assert!(max_versions(dir.path()).is_err());
    }
}
