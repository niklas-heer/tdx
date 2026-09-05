use crate::{document::Document, store::Store};
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
        let doc = Document::parse(store.baseline.clone().unwrap_or_default())?;
        Ok(Self {
            doc,
            store,
            past: VecDeque::new(),
            readonly,
            dirty: false,
        })
    }
    pub fn apply(&mut self, op: &str, index: usize, text: &str) -> Result<(), String> {
        if self.readonly || self.doc.readonly {
            return Err("read-only: editing is disabled".into());
        }
        let next = self.doc.change(op, index, text)?;
        if self.past.len() == 100 {
            self.past.pop_front();
        }
        self.past.push_back(self.doc.source.clone());
        self.doc = next;
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
        if self.readonly || self.doc.readonly {
            return Err("read-only: editing is disabled".into());
        }
        if let Some(source) = self.past.back() {
            let next = Document::parse(source.clone())?;
            self.past.pop_back();
            self.doc = next;
            self.save()?;
        }
        Ok(())
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
            assert!(editor.apply("edit", 0, "").is_err());
        }
        assert_eq!(editor.past.len(), 1);
        editor.undo().unwrap();
        assert!(editor.doc.tasks.is_empty());
    }
}
