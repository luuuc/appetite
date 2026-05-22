---
slug: workflow-package-skeleton
pitch: 01-02-workflow-engine-cli
cycle: "01"
hill: downhill
progress: 100
hill_updated_at: 2026-05-22T16:21:47Z
---
`internal/workflow/` with one file per loop operation (`shape.go`, `bet.go`, `cut.go`, `hill.go`, `cooldown.go`, `cycle.go`, `pass.go`, `signal.go`, `status.go`). Each exposes a `func Xxx(s store.Store, req XxxRequest) (XxxResult, error)`. Typed errors: `ErrInvalidTransition`, `ErrNotFound`, `ErrDoneCriteriaUnmet`.
