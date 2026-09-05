## ADDED Requirements
### Requirement: Evidence-based Rust evaluation
The project SHALL provide an isolated Rust prototype and Go comparison probes, shared correctness fixtures, reproducible benchmarks, Go CPU/allocation profiles, and a report recording toolchains, host, methodology, results, and feature gaps. The application SHALL remain Go and pre-1.0. Benchmark artifacts SHALL be ignored build outputs; runs SHALL never edit user todo files.
#### Scenario: Maintainer evaluates a rewrite
- **WHEN** a maintainer runs the documented evaluation task
- **THEN** correctness checks SHALL precede timing comparisons
- **AND** results SHALL distinguish comparable parser work from unequal application feature scope
- **AND** the report SHALL identify untested platforms and missing Rust persistence/TUI/history guarantees
