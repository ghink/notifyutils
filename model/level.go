package model

// Level classifies a message so that routing can decide which drivers receive it.
// It carries no other meaning for drivers: no driver behaves differently by level.
type Level = string

const (
	LevelDebug    Level = "debug"
	LevelInfo     Level = "info"
	LevelWarn     Level = "warn"
	LevelError    Level = "error"
	LevelCritical Level = "critical"
)
