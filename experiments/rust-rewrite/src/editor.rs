use crate::{actions::Action, document::Document, store::Store};
use std::collections::VecDeque;

pub struct Editor {
    pub doc: Document,
    pub store: Store,
    past: VecDeque<String>,
    pub readonly: bool,
    pub dirty: bool,
}
impl Editor {
    pub fn new(store: Store, readonly: bool) -> Result<Self, String> {
        let doc = Document::parse(
            store
                .baseline
                .clone()
                .unwrap_or_else(|| "# Todos\n\n".into()),
        )?;
        Ok(Self {
            doc,
            store,
            past: VecDeque::new(),
            readonly,
            dirty: false,
        })
    }
    pub fn apply(&mut self, op: &str, index: usize, text: &str) -> Result<(), String> {
        self.action(&Action::new(op, index, text)).map(|_| ())
    }
    pub fn action(&mut self, action: &Action) -> Result<usize, String> {
        let (next, index) = self.doc.apply(action)?;
        self.commit(next)?;
        Ok(index)
    }
    pub fn commit(&mut self, next: Document) -> Result<(), String> {
        if next.source == self.doc.source {
            return Ok(());
        }
        if self.past.len() == 100 {
            self.past.pop_front();
        }
        self.past.push_back(self.doc.source.clone());
        self.doc = next;
        self.dirty = true;
        if self.readonly { Ok(()) } else { self.save() }
    }
    pub fn manual_save(&mut self) -> Result<(), String> {
        self.save()
    }
    fn save(&mut self) -> Result<(), String> {
        self.dirty = true;
        match self.store.save(&self.doc.source) {
            Ok(()) => {
                self.dirty = false;
                Ok(())
            }
            Err(error) => {
                self.dirty = !error.committed;
                Err(error.message)
            }
        }
    }
    pub fn undo(&mut self) -> Result<(), String> {
        if let Some(source) = self.past.back() {
            let next = Document::parse(source.clone())?;
            self.past.pop_back();
            self.doc = next;
            self.dirty = true;
            if !self.readonly {
                self.save()?;
            }
        }
        Ok(())
    }
    pub fn force_save(&mut self) -> Result<(), String> {
        match self.store.force_save(&self.doc.source) {
            Ok(()) => {
                self.dirty = false;
                Ok(())
            }
            Err(error) => {
                self.dirty = !error.committed;
                Err(error.message)
            }
        }
    }
    pub fn restore(&mut self, id: i64) -> Result<(), String> {
        if self.readonly {
            return Err("read-only: restore is disabled".into());
        }
        let next = Document::parse(self.store.version(id)?)?;
        let saved = self.store.save(&next.source);
        if let Err(error) = &saved
            && !error.committed
        {
            return Err(error.message.clone());
        }
        if self.past.len() == 100 {
            self.past.pop_front();
        }
        self.past.push_back(self.doc.source.clone());
        self.doc = next;
        self.dirty = false;
        saved.map_err(|error| error.message)
    }
    pub fn reload(&mut self) -> Result<(), String> {
        let source = self.store.reload()?;
        self.doc = Document::parse(source)?;
        self.past.clear();
        self.dirty = false;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn undo_uses_current_revision_and_retains_conflicting_candidate() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("t.md");
        let original = "- [ ] café\n";
        std::fs::write(&path, original).unwrap();
        let store = Store::with_lock_root(&path, dir.path().join("locks")).unwrap();
        let mut editor = Editor::new(store, false).unwrap();
        editor.apply("toggle", 0, "").unwrap();
        editor.undo().unwrap();
        assert_eq!(std::fs::read_to_string(&path).unwrap(), original);
        std::fs::write(&path, "external").unwrap();
        assert!(
            editor
                .apply("edit", 0, "local")
                .unwrap_err()
                .contains("changed externally")
        );
        assert!(editor.dirty);
        assert_eq!(editor.doc.tasks[0].text, "local");
        assert_eq!(std::fs::read_to_string(&path).unwrap(), "external");
        editor.reload().unwrap();
        assert!(!editor.dirty);
    }
    #[test]
    fn invalid_edits_do_not_evict_undo() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("t.md");
        let store = Store::with_lock_root(&path, dir.path().join("locks")).unwrap();
        let mut editor = Editor::new(store, false).unwrap();
        editor.apply("add", 0, "one").unwrap();
        for _ in 0..110 {
            assert!(editor.apply("edit", 0, "\0").is_err());
        }
        assert_eq!(editor.past.len(), 1);
        editor.undo().unwrap();
        assert_eq!(editor.doc.tasks, Vec::<crate::document::Task>::new());
    }
}

#[cfg(test)]
mod recovery_tests {
    use super::*;
    #[test]
    fn restore_conflict_preserves_model_and_postcommit_error_advances_it() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("t.md");
        std::fs::write(&path, "- [ ] original\n").unwrap();
        let mut store = Store::with_lock_root(&path, dir.path().join("locks")).unwrap();
        store.enable_history(true).unwrap();
        let id = store.versions().unwrap()[0].id;
        let mut editor = Editor::new(store, false).unwrap();
        editor.apply("edit", 0, "local").unwrap();
        std::fs::write(&path, "- [ ] external\n").unwrap();
        assert!(editor.restore(id).unwrap_err().contains("externally"));
        assert_eq!(editor.doc.tasks[0].text, "local");
        assert!(!editor.dirty);
        assert_eq!(std::fs::read_to_string(&path).unwrap(), "- [ ] external\n");
        editor.reload().unwrap();
        let db = rusqlite::Connection::open(dir.path().join("versions.sqlite")).unwrap();
        db.execute_batch("CREATE TRIGGER fail_history BEFORE INSERT ON file_versions BEGIN SELECT RAISE(FAIL, 'injected history failure'); END;").unwrap();
        assert!(
            editor
                .restore(id)
                .unwrap_err()
                .starts_with("file saved, but")
        );
        assert_eq!(editor.doc.tasks[0].text, "original");
        assert!(!editor.dirty);
        assert_eq!(editor.store.baseline.as_deref(), Some("- [ ] original\n"));
        db.execute_batch("DROP TRIGGER fail_history;").unwrap();
        editor.store.finish().unwrap();
    }
    #[test]
    fn force_save_captures_overwritten_content_and_fails_closed() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("t.md");
        std::fs::write(&path, "- [ ] original\n").unwrap();
        let mut store = Store::with_lock_root(&path, dir.path().join("locks")).unwrap();
        store.enable_history(true).unwrap();
        let mut editor = Editor::new(store, false).unwrap();
        std::fs::write(&path, "- [ ] external\n").unwrap();
        assert!(editor.apply("edit", 0, "candidate").is_err());
        let db = rusqlite::Connection::open(dir.path().join("versions.sqlite")).unwrap();
        db.execute_batch("CREATE TRIGGER fail_history BEFORE INSERT ON file_versions BEGIN SELECT RAISE(FAIL, 'injected history failure'); END;").unwrap();
        assert!(
            editor
                .force_save()
                .unwrap_err()
                .contains("capture overwritten")
        );
        assert_eq!(std::fs::read_to_string(&path).unwrap(), "- [ ] external\n");
        assert!(editor.dirty);
        db.execute_batch("DROP TRIGGER fail_history;").unwrap();
        editor.force_save().unwrap();
        assert!(!editor.dirty);
        let ids = editor.store.versions().unwrap();
        let contents: Vec<_> = ids
            .iter()
            .map(|v| editor.store.version(v.id).unwrap())
            .collect();
        assert!(contents.contains(&"- [ ] external\n".to_owned()));
        assert!(contents.contains(&"# Todos\n\n- [ ] candidate\n".to_owned()));
        editor.store.finish().unwrap();
    }
}
