//! Save ordering shared by native I/O and deterministic fault simulation.
//! The driver supplies monotonic milliseconds and executes exactly one effect.
use serde::{Deserialize, Serialize};

#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub enum Phase {
    Prepare,
    Lock,
    Validate,
    CaptureBefore,
    Replace,
    SyncDirectory,
    CaptureAfter,
    Unlock,
    Done,
}
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub enum Completion {
    Ok,
    Busy,
    Failed(String),
}
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct Outcome {
    pub committed: bool,
    pub durable: bool,
    pub error: Option<String>,
}
#[derive(Debug)]
pub struct Save {
    phase: Phase,
    force: bool,
    locked: bool,
    lock_started: u64,
    outcome: Outcome,
}
impl Save {
    pub const fn new(force: bool) -> Self {
        Self {
            phase: Phase::Prepare,
            force,
            locked: false,
            lock_started: 0,
            outcome: Outcome {
                committed: false,
                durable: false,
                error: None,
            },
        }
    }
    pub const fn phase(&self) -> Phase {
        self.phase
    }
    pub const fn outcome(&self) -> &Outcome {
        &self.outcome
    }
    /// Complete the pending effect. Busy is permitted only for lock acquisition.
    pub fn advance(&mut self, result: Completion, now: u64) {
        if self.phase == Phase::Done {
            return;
        }
        let error = match result {
            Completion::Ok => None,
            Completion::Busy if self.phase == Phase::Lock => {
                if now.saturating_sub(self.lock_started) < 2000 {
                    return;
                }
                Some("file is busy in another tdx process".to_owned())
            }
            Completion::Busy => Some("unexpected busy I/O completion".to_owned()),
            Completion::Failed(error) => Some(error),
        };
        if let Some(error) = error {
            self.fail(&error);
            return;
        }
        self.phase = match self.phase {
            Phase::Prepare => {
                self.lock_started = now;
                Phase::Lock
            }
            Phase::Lock => {
                self.locked = true;
                Phase::Validate
            }
            Phase::Validate => {
                if self.force {
                    Phase::CaptureBefore
                } else {
                    Phase::Replace
                }
            }
            Phase::CaptureBefore => Phase::Replace,
            Phase::Replace => {
                self.outcome.committed = true;
                Phase::SyncDirectory
            }
            Phase::SyncDirectory => {
                self.outcome.durable = true;
                Phase::CaptureAfter
            }
            Phase::CaptureAfter => Phase::Unlock,
            Phase::Unlock | Phase::Done => {
                self.locked = false;
                Phase::Done
            }
        };
    }
    fn fail(&mut self, error: &str) {
        let label = match self.phase {
            Phase::SyncDirectory => "directory sync: ",
            Phase::CaptureAfter => "version capture: ",
            Phase::Unlock => "unlock: ",
            _ => "",
        };
        let detail = format!("{label}{error}");
        if let Some(previous) = &mut self.outcome.error {
            previous.push_str("; ");
            previous.push_str(&detail);
        } else {
            self.outcome.error = Some(if self.outcome.committed {
                format!("file saved, but {detail}")
            } else {
                detail
            });
        }
        self.phase = match self.phase {
            Phase::SyncDirectory => Phase::CaptureAfter,
            Phase::CaptureAfter => Phase::Unlock,
            Phase::Unlock => {
                self.locked = false;
                Phase::Done
            }
            _ if self.locked => Phase::Unlock,
            _ => Phase::Done,
        };
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn postcommit_failure_continues_cleanup_and_reports_commit() {
        let mut save = Save::new(false);
        for _ in 0..4 {
            save.advance(Completion::Ok, 0);
        }
        assert_eq!(save.phase(), Phase::SyncDirectory);
        save.advance(Completion::Failed("offline".into()), 0);
        assert_eq!(save.phase(), Phase::CaptureAfter);
        save.advance(Completion::Failed("database unavailable".into()), 0);
        save.advance(Completion::Ok, 0);
        assert_eq!(save.phase(), Phase::Done);
        assert!(save.outcome().committed);
        assert!(!save.outcome().durable);
        assert!(
            save.outcome()
                .error
                .as_ref()
                .unwrap()
                .contains("file saved")
        );
    }
    #[test]
    fn lock_timeout_uses_virtual_time_and_conflict_unlocks() {
        let mut save = Save::new(false);
        save.advance(Completion::Ok, 10);
        save.advance(Completion::Busy, 2009);
        assert_eq!(save.phase(), Phase::Lock);
        save.advance(Completion::Busy, 2010);
        assert_eq!(save.phase(), Phase::Done);
        assert!(!save.outcome().committed);
        let mut save = Save::new(false);
        save.advance(Completion::Ok, 0);
        save.advance(Completion::Ok, 0);
        save.advance(Completion::Failed("stale revision".into()), 0);
        assert_eq!(save.phase(), Phase::Unlock);
        save.advance(Completion::Ok, 0);
        assert!(!save.outcome().committed);
    }
}
