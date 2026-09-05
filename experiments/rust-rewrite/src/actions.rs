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
            index: index as isize,
            text: text.into(),
            ..Self::default()
        }
    }
}
