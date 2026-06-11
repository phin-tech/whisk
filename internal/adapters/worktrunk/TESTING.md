# Worktrunk Adapter Coverage Plan

This package is the boundary around the external `wt` CLI. Keep tests focused on
real command shapes, JSON drift, and destructive-operation safety.

## Covered Contracts

- Existing branch fallback relist failure returns `NotFoundError`.
- Existing branch fallback command failure preserves the fallback failure.
- Branch-exists stderr false positives do not trigger fallback.
- `wt list` tolerates boolean or integer working tree counts.
- `wt list` tolerates missing optional objects, `worktree: null`, missing path,
  and unknown future fields.
- Existing branch entries without a path are ignored for create binding.
- Dirty/locked remove stderr variants map to `DirtyError` or `LockedError`.
- Main/current worktrees are rejected before invoking `wt remove`.
- Cleaned path matching handles trivial stored-path differences.
- Override paths that are directories are treated as unavailable.
- Version command nonzero results are treated as unavailable.
- Existing-branch fallback omits `--base`.
- Backend list mapping covers every exposed `app.Worktree` field.

## Remaining Gaps

1. Remove after worktree already gone
   - Cover repeated remove/cleanup attempts.
   - Decide explicitly whether missing path is idempotent success or a
     `NotFoundError`. Current behavior is `NotFoundError`.

2. Path matching normalization
   - Symlink resolution should be explicit if we decide to support it.
   - Cross-platform case sensitivity should remain platform-native unless we
     intentionally support remote/non-native paths.

3. Base branch behavior
   - Cover that `--base` is passed only on create.
   - Existing-branch fallback must not pass `--base`; base has no meaning when
     switching to an existing branch.

4. Detection drift
    - Cover version command exits nonzero with stderr.
    - Cover prerelease/build metadata if `wt` starts emitting it.
    - Cover exact Windows terminal alias shapes if cross-platform support
      matters.

5. Backend field mapping
    - Cover mapping for `Dirty`, `Locked`, `IsMain`, `IsCurrent`, `Kind`,
      `Branch`, and `Path`.
    - The UI depends on these fields for disabling dangerous actions and showing
      useful worktree state.

6. UI/API behavior
    - Worktree generation should be disabled or no-op when an item is already
      bound.
    - Binding an existing worktree path should be treated as success, not as a
      create failure.

## Recommended Next Tests

Do these next:

1. Decide and test idempotent remove behavior.
2. Prerelease/build metadata if `wt` starts emitting it.
3. Symlink path matching if the UI stores resolved paths and `wt` returns link
   paths, or the reverse.
