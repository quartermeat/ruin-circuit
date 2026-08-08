# Agent Instructions

When an agent wakes up or begins a new work session in this repository, read `README.md` completely before taking the first action. It is required wake-up reading and contains the current project direction, design constraints, controls, progression model, and initial scope. It does not need to be reread for every prompt during the same session unless it has changed or a task requires refreshing its details.

Keep implementation decisions consistent with the README. Update the README when the project direction or agreed design changes materially.

Every commit must update `README.md` as a merged source of truth: preserve what is already documented, incorporate decisions made in the conversation, and keep the remaining planned work clearly represented.

Use the project versioning rules: an architectural refactor increments the first number (`vX.0.0`), a meaningful feature increments the second number (`vX.x.0`), and every build, fix, or tuning pass increments the third number (`vX.x.x`). The first number is not reserved for release readiness. Every push must be tagged with the matching version.
