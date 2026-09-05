#[cfg(windows)]
fn persist(temp: tempfile::NamedTempFile, target: &Path) -> std::io::Result<()> {
    use std::os::windows::ffi::OsStrExt;
    use windows_sys::Win32::Storage::FileSystem::{
        FILE_ATTRIBUTE_NORMAL, MOVEFILE_REPLACE_EXISTING, MOVEFILE_WRITE_THROUGH, MoveFileExW,
        SetFileAttributesW,
    };
    let path = temp.into_temp_path(); // Close the handle before replacement.
    let from: Vec<u16> = path.as_os_str().encode_wide().chain(Some(0)).collect();
    let to: Vec<u16> = target.as_os_str().encode_wide().chain(Some(0)).collect();
    // SAFETY: Both paths are nul-terminated, live UTF-16 buffers; no pointers escape.
    unsafe {
        if SetFileAttributesW(from.as_ptr(), FILE_ATTRIBUTE_NORMAL) == 0 {
            return Err(std::io::Error::last_os_error());
        }
        if MoveFileExW(
            from.as_ptr(),
            to.as_ptr(),
            MOVEFILE_REPLACE_EXISTING | MOVEFILE_WRITE_THROUGH,
        ) == 0
        {
            return Err(std::io::Error::last_os_error());
        }
    }
    Ok(())
}

#[cfg(test)]
use crate::history;
use crate::history::{History, Version};
use sha2::{Digest, Sha256};
use std::{
    fs::{self, OpenOptions},
    io::Write,
    path::{Path, PathBuf},
    thread,
    time::{Duration, Instant},
};

#[derive(Debug)]
pub struct SaveError {
    pub message: String,
    pub committed: bool,
}
impl From<std::io::Error> for SaveError {
    fn from(error: std::io::Error) -> Self {
        Self {
            message: error.to_string(),
            committed: false,
        }
    }
}
impl From<String> for SaveError {
    fn from(message: String) -> Self {
        Self {
            message,
            committed: false,
        }
    }
}

pub fn canonical(path: &Path) -> Result<PathBuf, String> {
    if path.symlink_metadata().is_ok() {
        return path
            .canonicalize()
            .map(normalize_path)
            .map_err(|e| e.to_string());
    }
    let absolute = std::path::absolute(path).map_err(|e| e.to_string())?;
    let parent = absolute
        .parent()
        .ok_or("file needs parent")?
        .canonicalize()
        .map_err(|e| e.to_string())?;
    Ok(normalize_path(
        parent.join(absolute.file_name().ok_or("file needs name")?),
    ))
}

fn read(path: &Path) -> Result<Option<String>, String> {
    match fs::metadata(path) {
        Ok(meta) if !meta.is_file() => Err("target must be a regular file".into()),
        Ok(_) => fs::read_to_string(path)
            .map(Some)
            .map_err(|e| e.to_string()),
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(None),
        Err(e) => Err(e.to_string()),
    }
}

pub struct Store {
    path: PathBuf,
    target: PathBuf,
    pub baseline: Option<String>,
    lock_root: PathBuf,
    history: Option<History>,
}
impl Store {
    pub fn load(path: &Path) -> Result<Self, String> {
        Self::with_lock_root(path, crate::config::directory()?.join("locks"))
    }
    pub fn with_lock_root(path: &Path, lock_root: PathBuf) -> Result<Self, String> {
        let target = canonical(path)?;
        let baseline = read(&target)?;
        Ok(Self {
            path: path.to_owned(),
            target,
            baseline,
            lock_root,
            history: None,
        })
    }
    #[cfg(test)]
    pub fn enable_history(&mut self, capture_loaded: bool) -> Result<(), String> {
        let dir = self
            .lock_root
            .parent()
            .ok_or("history directory unavailable")?;
        let max = history::max_versions(dir)?;
        self.enable_history_with_limit(capture_loaded, max)
    }
    pub fn enable_history_with_limit(
        &mut self,
        capture_loaded: bool,
        max: i64,
    ) -> Result<(), String> {
        let dir = self
            .lock_root
            .parent()
            .ok_or("history directory unavailable")?;
        let mut history = History::open(dir, max)?;
        if capture_loaded
            && let Some(source) = &self.baseline
            && let Err(error) = history.capture(&self.target, source)
        {
            let _ = history.close();
            return Err(format!("capture opened version: {error}"));
        }
        self.history = Some(history);
        Ok(())
    }
    pub fn versions(&mut self) -> Result<Vec<Version>, String> {
        self.history
            .as_mut()
            .ok_or("history is disabled")?
            .list(&self.target)
    }
    pub fn version(&mut self, id: i64) -> Result<String, String> {
        self.history
            .as_mut()
            .ok_or("history is disabled")?
            .read(&self.target, id)
    }
    pub fn finish(&mut self) -> Result<(), String> {
        self.history.take().map_or(Ok(()), |history| {
            history
                .close()
                .map_err(|e| format!("history shutdown failed: {e}"))
        })
    }
    pub fn changed(&self) -> Result<bool, String> {
        Ok(canonical(&self.path)? != self.target || read(&self.target)? != self.baseline)
    }
    pub fn reload(&mut self) -> Result<String, String> {
        let target = canonical(&self.path)?;
        let baseline = read(&target)?;
        let source = baseline.clone().unwrap_or_default();
        // Validate before replacing the revision known by the editor.
        // A rejected parse must not advance the revision used by guarded saves.
        crate::document::Document::parse(source.clone())?;
        if let (Some(history), Some(content)) = (&mut self.history, &baseline) {
            history
                .capture(&target, content)
                .map_err(|e| format!("capture reloaded version: {e}"))?;
        }
        self.target = target;
        self.baseline = baseline;
        Ok(source)
    }
    pub fn save(&mut self, source: &str) -> Result<(), SaveError> {
        self.save_with_hook(source, false, || {})
    }
    pub fn force_save(&mut self, source: &str) -> Result<(), SaveError> {
        self.save_with_hook(source, true, || {})
    }
    fn save_with_hook(
        &mut self,
        source: &str,
        force: bool,
        before_validation: impl FnOnce(),
    ) -> Result<(), SaveError> {
        if canonical(&self.path)? != self.target {
            return Err("file changed externally: target changed".to_owned().into());
        }
        let permissions = match fs::metadata(&self.target) {
            Ok(meta) => {
                if !meta.is_file() {
                    return Err("target must be a regular file".to_owned().into());
                }
                if meta.permissions().readonly() {
                    return Err("file is read-only".to_owned().into());
                }
                Some(meta.permissions())
            }
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => None,
            Err(e) => return Err(e.into()),
        };
        let parent = self
            .target
            .parent()
            .ok_or_else(|| "target has no parent directory".to_owned())?;
        let mut temp = tempfile::NamedTempFile::new_in(parent)?;
        if let Some(permissions) = permissions {
            temp.as_file().set_permissions(permissions)?;
        }
        temp.write_all(source.as_bytes())?;
        temp.as_file().sync_all()?;
        fs::create_dir_all(&self.lock_root)?;
        // Match the Go lock name and canonical path so cooperating Go/Rust saves coordinate.
        let digest = Sha256::digest(self.target.as_os_str().as_encoded_bytes());
        let name = format!("{digest:x}.lock");
        let lock = OpenOptions::new()
            .create(true)
            .truncate(false)
            .read(true)
            .write(true)
            .open(self.lock_root.join(name))?;
        let started = Instant::now();
        loop {
            match lock.try_lock() {
                Ok(()) => break,
                Err(std::fs::TryLockError::WouldBlock)
                    if started.elapsed() < Duration::from_secs(2) =>
                {
                    thread::sleep(Duration::from_millis(10));
                }
                Err(std::fs::TryLockError::WouldBlock) => {
                    return Err("file is busy in another tdx process".to_owned().into());
                }
                Err(std::fs::TryLockError::Error(e)) => return Err(e.into()),
            }
        }
        before_validation();
        let current = read(&self.target)?;
        if canonical(&self.path)? != self.target || (!force && current != self.baseline) {
            return Err("file changed externally; reload before saving"
                .to_owned()
                .into());
        }
        if force {
            let history = self
                .history
                .as_mut()
                .ok_or_else(|| "force-save requires version history".to_owned())?;
            if let Some(content) = &current {
                history
                    .capture(&self.target, content)
                    .map_err(|e| format!("capture overwritten version: {e}"))?;
            }
        }
        // The tempfile stays on the same filesystem. persist atomically replaces
        // the target on Unix; dropping the lock releases it on every return path.
        persist(temp, &self.target)?;
        self.baseline = Some(source.to_owned());
        let mut errors = Vec::new();
        if let Err(error) = sync_directory(parent) {
            errors.push(format!("directory sync: {error}"));
        }
        if let Some(history) = &mut self.history
            && let Err(error) = history.capture(&self.target, source)
        {
            errors.push(format!("version capture: {error}"));
        }
        if let Err(error) = lock.unlock() {
            errors.push(format!("unlock: {error}"));
        }
        if !errors.is_empty() {
            return Err(SaveError {
                message: format!("file saved, but {}", errors.join("; ")),
                committed: true,
            });
        }
        Ok(())
    }
}

#[cfg(unix)]
fn sync_directory(path: &Path) -> std::io::Result<()> {
    fs::File::open(path)?.sync_all()
}

// Go's filepath.EvalSymlinks returns ordinary Windows paths. Rust canonicalize
// uses extended prefixes; normalize identity before history keys and lock hashes.
#[allow(
    clippy::missing_const_for_fn,
    reason = "Windows path normalization allocates and cannot be const"
)]
fn normalize_path(path: PathBuf) -> PathBuf {
    #[cfg(windows)]
    {
        let text = path.to_string_lossy();
        if let Some(tail) = text.strip_prefix(r"\\?\UNC\") {
            return PathBuf::from(format!(r"\\{tail}"));
        }
        if let Some(tail) = text.strip_prefix(r"\\?\") {
            return PathBuf::from(tail);
        }
    }
    path
}
#[cfg(not(windows))]
fn persist(temp: tempfile::NamedTempFile, target: &Path) -> std::io::Result<()> {
    temp.persist(target).map(|_| ()).map_err(|e| e.error)
}
#[cfg(windows)]
#[expect(
    clippy::unnecessary_wraps,
    reason = "Matches the fallible Unix directory-sync interface at the shared save call site"
)]
const fn sync_directory(_: &Path) -> std::io::Result<()> {
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn conflict_preserves_external_bytes_and_cleans_temp() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("tasks.md");
        fs::write(&path, "- [ ] Old\n").unwrap();
        let mut store = Store::with_lock_root(&path, dir.path().join("locks")).unwrap();
        let error = store
            .save_with_hook("- [x] Old\n", false, || {
                fs::write(&path, "external").unwrap();
            })
            .unwrap_err();
        assert!(!error.committed);
        assert!(error.message.contains("changed externally"));
        assert_eq!(fs::read_to_string(&path).unwrap(), "external");
        assert_eq!(fs::read_dir(dir.path()).unwrap().count(), 2);
    }
    #[test]
    fn rejected_reload_keeps_original_revision() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("tasks.md");
        fs::write(&path, "- [ ] old\n").unwrap();
        let mut store = Store::with_lock_root(&path, dir.path().join("locks")).unwrap();
        fs::write(&path, [0xff]).unwrap();
        assert!(store.reload().is_err());
        assert_eq!(store.baseline.as_deref(), Some("- [ ] old\n"));
        assert!(store.save("- [x] old\n").is_err());
    }
    #[test]
    fn missing_and_empty_are_distinct_revisions() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("tasks.md");
        let mut store = Store::with_lock_root(&path, dir.path().join("locks")).unwrap();
        fs::write(&path, "").unwrap();
        assert!(store.save("new").is_err());
    }
    #[cfg(unix)]
    #[test]
    fn symlink_mode_and_readonly() {
        use std::os::unix::fs::{PermissionsExt, symlink};
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("tasks.md");
        fs::write(&path, "old").unwrap();
        fs::set_permissions(&path, fs::Permissions::from_mode(0o640)).unwrap();
        let alias = dir.path().join("alias.md");
        symlink(&path, &alias).unwrap();
        let mut store = Store::with_lock_root(&alias, dir.path().join("locks")).unwrap();
        store.save("new").unwrap();
        assert!(alias.is_symlink());
        assert_eq!(fs::read_to_string(&path).unwrap(), "new");
        assert_eq!(
            fs::metadata(&path).unwrap().permissions().mode() & 0o777,
            0o640
        );
        fs::set_permissions(&path, fs::Permissions::from_mode(0o444)).unwrap();
        assert!(store.save("bad").is_err());
        assert_eq!(fs::read_to_string(&path).unwrap(), "new");
    }
    #[cfg(unix)]
    #[test]
    fn symlink_retarget_is_a_conflict() {
        use std::os::unix::fs::symlink;
        let dir = tempfile::tempdir().unwrap();
        let a = dir.path().join("a");
        let b = dir.path().join("b");
        let alias = dir.path().join("alias");
        fs::write(&a, "same").unwrap();
        fs::write(&b, "same").unwrap();
        symlink(&a, &alias).unwrap();
        let mut store = Store::with_lock_root(&alias, dir.path().join("locks")).unwrap();
        fs::remove_file(&alias).unwrap();
        symlink(&b, &alias).unwrap();
        assert!(store.save("bad").is_err());
        assert_eq!(fs::read_to_string(&a).unwrap(), "same");
        assert_eq!(fs::read_to_string(&b).unwrap(), "same");
    }
}
