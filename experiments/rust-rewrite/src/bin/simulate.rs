//! Development-only deterministic save simulator. Never opens user documents.
#[path = "../save_protocol.rs"]
mod save_protocol;
#[path = "../simulation.rs"]
mod simulation;
use simulation::Mutation;
use std::{collections::BTreeMap, env, fs, process::ExitCode};
fn run() -> Result<bool, String> {
    let mut args = env::args().skip(1);
    let (mut seed, mut runs, mut steps) = (0_u64, 1_u64, 200_usize);
    let mut trace = None;
    let mut mutation = Mutation::None;
    while let Some(arg) = args.next() {
        let value = args
            .next()
            .ok_or_else(|| format!("{arg} requires a value"))?;
        match arg.as_str() {
            "--seed" => seed = value.parse().map_err(|_| "invalid seed")?,
            "--runs" => runs = value.parse().map_err(|_| "invalid runs")?,
            "--steps" => steps = value.parse().map_err(|_| "invalid steps")?,
            "--trace" => trace = Some(value),
            "--mutation" => {
                mutation = match value.as_str() {
                    "skip-validation" => Mutation::SkipValidation,
                    "skip-sync" => Mutation::SkipSync,
                    "skip-replacement" => Mutation::SkipReplacement,
                    _ => return Err("unknown mutation".into()),
                }
            }
            _ => return Err(format!("unknown argument {arg}")),
        }
    }
    if runs == 0
        || runs > 100_000
        || steps == 0
        || steps > 100_000
        || (trace.is_some() && runs != 1)
    {
        return Err("runs/steps must be 1..100000; trace requires one run".into());
    }
    let mut counts: BTreeMap<String, u64> = BTreeMap::new();
    let mut hashes = Vec::new();
    for offset in 0..runs {
        let current = seed.checked_add(offset).ok_or("seed overflow")?;
        let report = simulation::run(current, steps, mutation);
        if let Some(path) = &trace {
            fs::write(
                path,
                serde_json::to_vec_pretty(&report).map_err(|e| e.to_string())?,
            )
            .map_err(|e| e.to_string())?;
        }
        if !report.failures.is_empty() {
            println!(
                "{}",
                serde_json::to_string(&report).map_err(|e| e.to_string())?
            );
            return Ok(false);
        }
        for (key, value) in report.counts {
            *counts.entry(key).or_default() += value;
        }
        hashes.push(report.trace_sha256);
    }
    println!(
        "{}",
        serde_json::json!({"schema":1,"seed":seed,"runs":runs,"steps":steps,"counts":counts,"trace_sha256":hashes})
    );
    Ok(true)
}
fn main() -> ExitCode {
    match run() {
        Ok(true) => ExitCode::SUCCESS,
        Ok(false) => ExitCode::FAILURE,
        Err(error) => {
            eprintln!("{error}");
            ExitCode::FAILURE
        }
    }
}
