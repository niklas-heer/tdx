//! Deterministic cooperative scheduling of the production save protocol.
//! Native filesystem/SQLite implementations are tested separately, not emulated here.
use crate::save_protocol::{Completion, Phase, Save};
use serde::Serialize;
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, BTreeSet};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Mutation {
    None,
    SkipValidation,
    SkipSync,
    SkipReplacement,
}
#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
pub struct Event {
    pub tick: u64,
    pub writer: usize,
    pub effect: String,
    pub result: String,
    pub visible_sha256: String,
    pub durable_sha256: String,
}
#[derive(Debug, Serialize)]
pub struct Report {
    pub schema: u32,
    pub seed: u64,
    pub steps: usize,
    pub counts: BTreeMap<String, u64>,
    pub failures: Vec<String>,
    pub trace_sha256: String,
    pub trace: Vec<Event>,
}
struct Random(u64);
impl Random {
    const fn next(&mut self) -> u64 {
        // Explicit wrapping is the specified SplitMix64 algorithm, identical on all hosts.
        self.0 = self.0.wrapping_add(0x9e37_79b9_7f4a_7c15);
        let mut z = self.0;
        z = (z ^ (z >> 30)).wrapping_mul(0xbf58_476d_1ce4_e5b9);
        z = (z ^ (z >> 27)).wrapping_mul(0x94d0_49bb_1331_11eb);
        z ^ (z >> 31)
    }
    const fn chance(&mut self, denominator: u64) -> bool {
        self.next().is_multiple_of(denominator)
    }
    const fn latency(&mut self) -> u64 {
        if self.chance(17) {
            3000
        } else {
            1 + self.next() % 50
        }
    }
}
struct Writer {
    save: Save,
    baseline: String,
    proposed: String,
    prepared: bool,
    captured: Option<String>,
    written: bool,
    force: bool,
    ready: u64,
}
struct World {
    visible: String,
    durable: String,
    lock: Option<usize>,
    history: BTreeSet<String>,
    known: BTreeSet<String>,
}
fn hash(source: &str) -> String {
    format!("{:x}", Sha256::digest(source.as_bytes()))
}
fn count(report: &mut Report, key: &str) {
    *report.counts.entry(key.into()).or_default() += 1;
}
fn record(report: &mut Report, world: &World, now: u64, writer: usize, effect: &str, result: &str) {
    report.trace.push(Event {
        tick: now,
        writer,
        effect: effect.into(),
        result: result.into(),
        visible_sha256: hash(&world.visible),
        durable_sha256: hash(&world.durable),
    });
}
fn writer(world: &mut World, serial: usize, force: bool, ready: u64) -> Writer {
    // Whole Markdown payloads make partial replacements independently observable.
    let proposed = format!(
        "---\ncustom: retained\n---\n# Project\n\n<div>opaque</div>\n\n- [ ] café 🦀 revision {serial}\n\n| a | b |\n|---|---|\n| x | y |\n"
    );
    world.known.insert(proposed.clone());
    Writer {
        save: Save::new(force),
        baseline: world.visible.clone(),
        proposed,
        prepared: false,
        captured: None,
        written: false,
        force,
        ready,
    }
}
fn effect(
    world: &mut World,
    actor: &mut Writer,
    id: usize,
    phase: Phase,
    fault: bool,
    mutation: Mutation,
) -> Completion {
    if fault {
        return Completion::Failed(
            match phase {
                Phase::Prepare => "torn temporary write / disk full",
                Phase::CaptureBefore | Phase::CaptureAfter => "history unavailable",
                Phase::SyncDirectory => "storage sync failed",
                _ => "storage unavailable",
            }
            .into(),
        );
    }
    match phase {
        Phase::Prepare => {
            actor.prepared = true;
        }
        Phase::Lock => {
            if world.lock.is_some() {
                return Completion::Busy;
            }
            world.lock = Some(id);
        }
        Phase::Validate => {
            if !actor.force
                && actor.baseline != world.visible
                && mutation != Mutation::SkipValidation
            {
                return Completion::Failed("file changed externally; reload before saving".into());
            }
        }
        Phase::CaptureBefore => {
            world.history.insert(world.visible.clone());
            actor.captured = Some(world.visible.clone());
        }
        Phase::Replace => {
            if mutation != Mutation::SkipReplacement {
                world.visible.clone_from(&actor.proposed);
                actor.written = true;
            }
        }
        Phase::SyncDirectory => {
            if mutation != Mutation::SkipSync {
                world.durable.clone_from(&world.visible);
            }
        }
        Phase::CaptureAfter => {
            world.history.insert(actor.proposed.clone());
        }
        Phase::Unlock => {
            world.lock = None;
        }
        Phase::Done => return Completion::Failed("completed writer scheduled".into()),
    }
    Completion::Ok
}
fn step(
    world: &mut World,
    actor: &mut Writer,
    id: usize,
    now: u64,
    fault: bool,
    mutation: Mutation,
    report: &mut Report,
) {
    let phase = actor.save.phase();
    let before = world.visible.clone();
    // Independent oracle checks the real preconditions, not the protocol's state flags.
    if phase == Phase::Replace && !fault {
        if world.lock != Some(id) || !actor.prepared {
            report
                .failures
                .push("replacement without lock and prepared bytes".into());
        }
        if !actor.force && actor.baseline != before {
            report
                .failures
                .push("stale writer overwrote a newer revision".into());
        }
        if actor.force && actor.captured.as_ref() != Some(&before) {
            report
                .failures
                .push("force-save omitted overwritten history".into());
        }
    }
    let completion = effect(world, actor, id, phase, fault, mutation);
    if completion != Completion::Ok && world.visible != before {
        report
            .failures
            .push("failed effect changed target bytes".into());
    }
    if !world.known.contains(&world.visible) {
        report
            .failures
            .push("target contains partial or unknown bytes".into());
    }
    count(report, &format!("phase/{phase:?}"));
    if fault {
        count(report, &format!("fault/{phase:?}"));
    }
    if completion == Completion::Busy {
        count(report, "lock_busy");
    }
    let label = format!("{completion:?}");
    actor.save.advance(completion, now);
    record(report, world, now, id, &format!("{phase:?}"), &label);
    if actor.save.phase() == Phase::Done {
        if actor.save.outcome().committed != actor.written {
            report
                .failures
                .push("reported commit disagrees with actual replacement".into());
        }
        if actor.save.outcome().error.is_none()
            && (!actor.written || world.durable != actor.proposed)
        {
            report
                .failures
                .push("acknowledged save is not durable".into());
        }
        count(
            report,
            if actor.written {
                "committed"
            } else {
                "rejected"
            },
        );
        // Native resource destruction releases a held lock even when unlock reports failure.
        if world.lock == Some(id) {
            world.lock = None;
        }
    }
}

#[expect(
    clippy::too_many_lines,
    reason = "Keep virtual scheduling, failure injection and recovery ordering together"
)]
pub fn run(seed: u64, steps: usize, mutation: Mutation) -> Report {
    let initial = "# Project\n\n- [ ] original café\n".to_owned();
    let mut world = World {
        visible: initial.clone(),
        durable: initial.clone(),
        lock: None,
        history: BTreeSet::new(),
        known: BTreeSet::from([initial]),
    };
    let mut rng = Random(seed);
    let mut report = Report {
        schema: 1,
        seed,
        steps,
        counts: BTreeMap::new(),
        failures: vec![],
        trace_sha256: String::new(),
        trace: vec![],
    };
    let mut actors: Vec<Option<Writer>> = (0..3)
        .map(|id| Some(writer(&mut world, id, false, rng.latency())))
        .collect();
    let mut now = 0;
    let mut serial = 3;
    for _ in 0..steps {
        for slot in &mut actors {
            if slot.is_none() {
                *slot = Some(writer(
                    &mut world,
                    serial,
                    rng.chance(7),
                    now + rng.latency(),
                ));
                serial += 1;
            }
        }
        let next = actors
            .iter()
            .enumerate()
            .filter_map(|(id, a)| a.as_ref().map(|a| (a.ready, id)))
            .min();
        let Some((ready, id)) = next else { break };
        now = ready;
        if rng.chance(101) {
            // Process death does not discard the OS page cache; power loss does.
            world.visible.clone_from(&world.durable);
            world.lock = None;
            world.history.clear();
            for actor in &mut actors {
                *actor = None;
            }
            count(&mut report, "power_loss");
            record(
                &mut report,
                &world,
                now,
                id,
                "PowerLoss",
                "discard unsynced state",
            );
            continue;
        }
        if rng.chance(29) {
            actors[id] = None;
            if world.lock == Some(id) {
                world.lock = None;
            }
            count(&mut report, "process_crash");
            record(
                &mut report,
                &world,
                now,
                id,
                "ProcessCrash",
                "lock released",
            );
            continue;
        }
        if world.lock.is_none() && rng.chance(19) {
            world.visible = format!("# External editor\n\n- [ ] external {serial}\n");
            serial += 1;
            world.durable.clone_from(&world.visible);
            world.known.insert(world.visible.clone());
            count(&mut report, "external_edit");
            record(
                &mut report,
                &world,
                now,
                id,
                "ExternalEdit",
                "durable external revision",
            );
        }
        if let Some(actor) = &mut actors[id] {
            let fault = rng.chance(13);
            step(&mut world, actor, id, now, fault, mutation, &mut report);
            actor.ready = now + rng.latency();
            if actor.save.phase() == Phase::Done {
                actors[id] = None;
            }
        }
        if !report.failures.is_empty() {
            break;
        }
    }
    // Stop faults, restart/reload and prove bounded progress through the same engine.
    if report.failures.is_empty() {
        world.lock = None;
        let mut recovered = writer(&mut world, serial, false, now);
        for _ in 0..12 {
            now += 1;
            step(
                &mut world,
                &mut recovered,
                0,
                now,
                false,
                mutation,
                &mut report,
            );
            if recovered.save.phase() == Phase::Done {
                break;
            }
        }
        if recovered.save.phase() != Phase::Done || recovered.save.outcome().error.is_some() {
            report
                .failures
                .push("no progress after faults stopped".into());
        } else if report.failures.is_empty() {
            let acknowledged = world.visible.clone();
            world.visible.clone_from(&world.durable);
            if world.visible != acknowledged {
                report
                    .failures
                    .push("successful recovery lost on power failure".into());
            }
            count(&mut report, "recovery_passed");
            record(
                &mut report,
                &world,
                now,
                0,
                "Recovery",
                "save and power-loss check",
            );
        }
    }
    // Digest excludes itself and uses stable ordered maps and an explicit trace schema.
    match serde_json::to_vec(&report.trace) {
        Ok(bytes) => report.trace_sha256 = format!("{:x}", Sha256::digest(bytes)),
        Err(error) => report.failures.push(error.to_string()),
    }
    report
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn seeds_replay_exactly_and_recover() {
        for seed in 0..32 {
            let first = run(seed, 200, Mutation::None);
            let second = run(seed, 200, Mutation::None);
            assert!(
                first.failures.is_empty(),
                "seed {seed}: {:?}",
                first.failures
            );
            assert_eq!(first.trace, second.trace);
            assert_eq!(first.trace_sha256, second.trace_sha256);
            assert_eq!(first.counts.get("recovery_passed"), Some(&1));
        }
    }
    #[test]
    fn oracle_detects_deliberately_broken_guarantees() {
        for mutation in [
            Mutation::SkipValidation,
            Mutation::SkipSync,
            Mutation::SkipReplacement,
        ] {
            assert!(
                (0..32).any(|seed| !run(seed, 200, mutation).failures.is_empty()),
                "undetected {mutation:?}"
            );
        }
    }
}
