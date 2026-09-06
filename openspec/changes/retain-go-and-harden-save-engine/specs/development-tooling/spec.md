## REMOVED Requirements
### Requirement: Evidence-based Rust evaluation
**Reason**: The user retired the Rust experiment; historical evidence remains in Git and archived changes.
**Migration**: Use the maintained Go application and deterministic Go engine checks.

### Requirement: Runnable Rust rewrite evaluation
**Reason**: The user retired the Rust experiment; historical evidence remains in Git and archived changes.
**Migration**: Use the maintained Go application and deterministic Go engine checks.

### Requirement: Reproducible application comparison
**Reason**: The user retired the Rust experiment; historical evidence remains in Git and archived changes.
**Migration**: Use the maintained Go application and deterministic Go engine checks.

### Requirement: Basic Rust terminal prototype
**Reason**: The user retired the Rust experiment; historical evidence remains in Git and archived changes.
**Migration**: Use the maintained Go application and deterministic Go engine checks.

### Requirement: History-enabled Rust evaluation
**Reason**: The user retired the Rust experiment; historical evidence remains in Git and archived changes.
**Migration**: Use the maintained Go application and deterministic Go engine checks.

### Requirement: Rust terminal recovery milestone
**Reason**: The user retired the Rust experiment; historical evidence remains in Git and archived changes.
**Migration**: Use the maintained Go application and deterministic Go engine checks.

### Requirement: Comparable history workloads
**Reason**: The user retired the Rust experiment; historical evidence remains in Git and archived changes.
**Migration**: Use the maintained Go application and deterministic Go engine checks.

### Requirement: Full-featured Rust comparison candidate
**Reason**: The user retired the Rust experiment; historical evidence remains in Git and archived changes.
**Migration**: Use the maintained Go application and deterministic Go engine checks.

### Requirement: Differential bug handling
**Reason**: The user retired the Rust experiment; historical evidence remains in Git and archived changes.
**Migration**: Use the maintained Go application and deterministic Go engine checks.

### Requirement: Parity-gated performance results
**Reason**: The user retired the Rust experiment; historical evidence remains in Git and archived changes.
**Migration**: Use the maintained Go application and deterministic Go engine checks.

### Requirement: Complete Ratatui presentation parity
**Reason**: The user retired the Rust experiment; historical evidence remains in Git and archived changes.
**Migration**: Use the maintained Go application and deterministic Go engine checks.

### Requirement: Hardened Rust development workflow
**Reason**: The user retired the Rust experiment; historical evidence remains in Git and archived changes.
**Migration**: Use the maintained Go application and deterministic Go engine checks.

### Requirement: Deterministic Rust save simulation
**Reason**: The user retired the Rust experiment; historical evidence remains in Git and archived changes.
**Migration**: Use the maintained Go application and deterministic Go engine checks.

### Requirement: Rust full-document Markdown editing
**Reason**: The user retired the Rust experiment; historical evidence remains in Git and archived changes.
**Migration**: Use the maintained Go application and deterministic Go engine checks.

## ADDED Requirements
### Requirement: Maintained Go application and engine evidence
Go SHALL be the sole maintained application. Setup, builds and CI SHALL not require Rust. Useful shared correctness cases SHALL remain executable Go regressions. A developer simulation command SHALL produce replayable fault traces and source/build identity without touching user task files.

#### Scenario: Go-only development
- **WHEN** a contributor sets up, builds or tests tdx
- **THEN** no Rust toolchain or rewrite is required
- **AND** local tasks and Linux, macOS and Windows CI exercise the save protocol, deterministic campaign and native persistence checks
