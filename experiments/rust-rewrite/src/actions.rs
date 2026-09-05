#![deny(
    clippy::indexing_slicing,
    clippy::arithmetic_side_effects,
    clippy::string_slice
)]
use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
#[serde(default)]
pub struct Action {
    pub kind: String,
    pub index: isize,
    pub target: isize,
    pub text: String,
    pub checked: bool,
    pub insert_after: bool,
    pub level: usize,
}
impl Action {
    pub fn new(kind: &str, index: usize, text: &str) -> Self {
        Self {
            kind: kind.into(),
            index: isize::try_from(index).unwrap_or(-1),
            text: text.into(),
            ..Self::default()
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oversized_index_is_rejected_without_wrapping_to_a_valid_task() {
        let action = Action::new("toggle", usize::MAX, "");
        assert_eq!(action.index, -1);
        let doc = crate::document::Document::parse("- [ ] unchanged\n".into()).unwrap();
        assert!(doc.apply(&action).is_err());
        assert!(!doc.tasks[0].checked);
    }
}
