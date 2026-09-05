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
        use crate::save_protocol::{Completion, Phase, Save};
        let started = Instant::now();
        let mut save = Save::new(force);
        let mut driver = NativeSave {
            store: self,
            source,
            force,
            temp: None,
            lock: None,
            current: None,
        };
        let mut hook = Some(before_validation);
        while save.phase() != Phase::Done {
            let phase = save.phase();
            if phase == Phase::Validate
                && let Some(hook) = hook.take()
            {
                hook();
            }
            let result = driver.execute(phase);
            if result == Completion::Busy {
                thread::sleep(Duration::from_millis(10));
            }
            save.advance(
                result,
                u64::try_from(started.elapsed().as_millis()).unwrap_or(u64::MAX),
            );
            #[cfg(test)]
            crash_hook(phase);
        }
        save.outcome().error.as_ref().map_or(Ok(()), |message| {
            Err(SaveError {
                message: message.clone(),
                committed: save.outcome().committed,
            })
        })
    }
}

struct NativeSave<'a> {
    store: &'a mut Store,
    source: &'a str,
    force: bool,
    temp: Option<tempfile::NamedTempFile>,
    lock: Option<fs::File>,
    current: Option<String>,
}
impl NativeSave<'_> {
    fn execute(&mut self, phase: crate::save_protocol::Phase) -> crate::save_protocol::Completion {
        use crate::save_protocol::{Completion, Phase};
        if phase == Phase::Lock {
            if self.lock.is_none() {
                match self.open_lock() {
                    Ok(lock) => self.lock = Some(lock),
                    Err(error) => return Completion::Failed(error.to_string()),
                }
            }
            return match self.lock.as_ref().map(fs::File::try_lock) {
                Some(Ok(())) => Completion::Ok,
                Some(Err(std::fs::TryLockError::WouldBlock)) => Completion::Busy,
                Some(Err(std::fs::TryLockError::Error(error))) => {
                    Completion::Failed(error.to_string())
                }
                None => Completion::Failed("lock handle unavailable".into()),
            };
        }
        match self.perform(phase) {
            Ok(()) => Completion::Ok,
            Err(error) => Completion::Failed(error),
        }
    }
    fn open_lock(&self) -> std::io::Result<fs::File> {
        fs::create_dir_all(&self.store.lock_root)?;
        let digest = Sha256::digest(self.store.target.as_os_str().as_encoded_bytes());
        OpenOptions::new()
            .create(true)
            .truncate(false)
            .read(true)
            .write(true)
            .open(self.store.lock_root.join(format!("{digest:x}.lock")))
    }
    fn prepare(&mut self) -> Result<(), String> {
        if canonical(&self.store.path)? != self.store.target {
            return Err("file changed externally: target changed".into());
        }
        let permissions = match fs::metadata(&self.store.target) {
            Ok(meta) => {
                if !meta.is_file() {
                    return Err("target must be a regular file".into());
                }
                if meta.permissions().readonly() {
                    return Err("file is read-only".into());
                }
                Some(meta.permissions())
            }
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => None,
            Err(error) => return Err(error.to_string()),
        };
        let parent = self
            .store
            .target
            .parent()
            .ok_or("target has no parent directory")?;
        let mut temp = tempfile::NamedTempFile::new_in(parent).map_err(|e| e.to_string())?;
        if let Some(permissions) = permissions {
            temp.as_file()
                .set_permissions(permissions)
                .map_err(|e| e.to_string())?;
        }
        temp.write_all(self.source.as_bytes())
            .map_err(|e| e.to_string())?;
        temp.as_file().sync_all().map_err(|e| e.to_string())?;
        self.temp = Some(temp);
        Ok(())
    }
    fn perform(&mut self, phase: crate::save_protocol::Phase) -> Result<(), String> {
        use crate::save_protocol::Phase;
        match phase {
            Phase::Prepare => self.prepare(),
            Phase::Validate => {
                self.current = read(&self.store.target)?;
                if canonical(&self.store.path)? != self.store.target
                    || (!self.force && self.current != self.store.baseline)
                {
                    Err("file changed externally; reload before saving".into())
                } else {
                    Ok(())
                }
            }
            Phase::CaptureBefore => {
                let history = self
                    .store
                    .history
                    .as_mut()
                    .ok_or("force-save requires version history")?;
                if let Some(content) = &self.current {
                    history
                        .capture(&self.store.target, content)
                        .map_err(|e| format!("capture overwritten version: {e}"))?;
                }
                Ok(())
            }
            Phase::Replace => {
                let temp = self.temp.take().ok_or("prepared replacement unavailable")?;
                persist(temp, &self.store.target).map_err(|e| e.to_string())?;
                self.store.baseline = Some(self.source.to_owned());
                Ok(())
            }
            Phase::SyncDirectory => sync_directory(
                self.store
                    .target
                    .parent()
                    .ok_or("target has no parent directory")?,
            )
            .map_err(|e| e.to_string()),
            Phase::CaptureAfter => {
                if let Some(history) = &mut self.store.history {
                    history.capture(&self.store.target, self.source)?;
                }
                Ok(())
            }
            Phase::Unlock => self
                .lock
                .as_ref()
                .ok_or("lock handle unavailable")?
                .unlock()
                .map_err(|e| e.to_string()),
            Phase::Lock | Phase::Done => Err("unexpected native save effect".into()),
        }
    }
}

#[cfg(test)]
fn crash_hook(phase: crate::save_protocol::Phase) {
    // Only the explicitly selected subprocess test can enable this barrier.
    if std::env::var("TDX_NATIVE_CRASH_CHILD").as_deref() != Ok("1") {
        return;
    }
    if std::env::var("TDX_NATIVE_CRASH_PHASE").ok().as_deref() == Some(&format!("{phase:?}")) {
        if let Ok(path) = std::env::var("TDX_NATIVE_CRASH_READY") {
            let _ = fs::write(path, "ready");
        }
        loop {
            thread::sleep(Duration::from_millis(10));
        }
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
    #[ignore = "subprocess barrier used by native crash/recovery test"]
    fn native_crash_child() {
        if std::env::var("TDX_NATIVE_CRASH_CHILD").as_deref() != Ok("1") {
            return;
        }
        let base = PathBuf::from(std::env::var("TDX_NATIVE_CRASH_BASE").unwrap());
        let mut store = Store::with_lock_root(&base.join("tasks.md"), base.join("locks")).unwrap();
        store.enable_history(true).unwrap();
        store.save("- [x] replacement café\n").unwrap();
    }
    #[test]
    fn native_crash_at_save_boundaries_preserves_complete_file_and_recovers() {
        for phase in [
            "Prepare",
            "Validate",
            "Replace",
            "SyncDirectory",
            "CaptureAfter",
        ] {
            let dir = tempfile::tempdir().unwrap();
            let path = dir.path().join("tasks.md");
            let ready = dir.path().join("ready");
            fs::write(&path, "- [ ] original\n").unwrap();
            let mut child = std::process::Command::new(std::env::current_exe().unwrap())
                .args([
                    "--exact",
                    "store::tests::native_crash_child",
                    "--ignored",
                    "--nocapture",
                ])
                .env("TDX_NATIVE_CRASH_CHILD", "1")
                .env("TDX_NATIVE_CRASH_BASE", dir.path())
                .env("TDX_NATIVE_CRASH_PHASE", phase)
                .env("TDX_NATIVE_CRASH_READY", &ready)
                .stdout(std::process::Stdio::null())
                .stderr(std::process::Stdio::null())
                .spawn()
                .unwrap();
            let started = Instant::now();
            while !ready.exists() && started.elapsed() < Duration::from_secs(10) {
                if child.try_wait().unwrap().is_some() {
                    break;
                }
                thread::sleep(Duration::from_millis(5));
            }
            let reached = ready.exists();
            let _ = child.kill();
            child.wait().unwrap();
            assert!(reached, "child did not reach {phase}");
            let expected = if matches!(phase, "Prepare" | "Validate") {
                "- [ ] original\n"
            } else {
                "- [x] replacement café\n"
            };
            assert_eq!(fs::read_to_string(&path).unwrap(), expected, "{phase}");
            let mut store = Store::with_lock_root(&path, dir.path().join("locks")).unwrap();
            store.enable_history(true).unwrap();
            store.save("- [ ] recovered\n").unwrap();
            store.finish().unwrap();
            assert_eq!(fs::read_to_string(&path).unwrap(), "- [ ] recovered\n");
        }
    }
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
