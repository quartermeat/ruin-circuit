# Agent Instructions

When an agent wakes up or begins a new work session in this repository, read `README.md` completely before taking the first action. It is required wake-up reading and contains the current project direction, design constraints, controls, progression model, and initial scope. It does not need to be reread for every prompt during the same session unless it has changed or a task requires refreshing its details.

Keep implementation decisions consistent with the README. Update the README when the project direction or agreed design changes materially.

Every commit must update `README.md` as a merged source of truth: preserve what is already documented, incorporate decisions made in the conversation, and keep the remaining planned work clearly represented.

Use the project versioning rules: every build increments the patch number (`v0.0.x`), a meaningful feature increments the minor number (`v0.x.0`), and `v1.0.0` or later is reserved for a release Jeremy considers shareable. Every push must be tagged with the matching version.
