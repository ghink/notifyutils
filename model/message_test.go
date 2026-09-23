package model

import (
	"testing"
)

func TestExtra(t *testing.T) {
	msg := Message{Extras: map[string]any{
		"string":   "value",
		"empty":    "",
		"strings":  []string{"a", "b"},
		"bool":     true,
		"int":      7,
		"int64":    int64(8),
		"fromJSON": float64(9),
		"nil":      nil,
	}}

	if msg.Extra("string") != any("value") {
		t.Errorf("Extra(string) = %v, want value", msg.Extra("string"))
	}
	if msg.Extra("missing") != nil {
		t.Errorf("Extra(missing) = %v, want nil", msg.Extra("missing"))
	}
}

func TestExtraString(t *testing.T) {
	msg := Message{Extras: map[string]any{"k": "v", "wrong": 1}}

	if got := msg.ExtraString("k"); got != "v" {
		t.Errorf("ExtraString(k) = %q, want v", got)
	}
	if got := msg.ExtraString("wrong"); got != "" {
		t.Errorf("ExtraString(wrong type) = %q, want empty", got)
	}
	if got := msg.ExtraString("absent"); got != "" {
		t.Errorf("ExtraString(absent) = %q, want empty", got)
	}
}

func TestExtraStrings(t *testing.T) {
	msg := Message{Extras: map[string]any{
		"list":    []string{"a", "b"},
		"single":  "only",
		"blank":   "",
		"nothing": nil,
	}}

	list := msg.ExtraStrings("list")
	if len(list) != 2 || list[0] != "a" || list[1] != "b" {
		t.Errorf("ExtraStrings(list) = %v, want [a b]", list)
	}
	// A lone value is the common case and should not force callers to wrap it.
	single := msg.ExtraStrings("single")
	if len(single) != 1 || single[0] != "only" {
		t.Errorf("ExtraStrings(single) = %v, want [only]", single)
	}
	if got := msg.ExtraStrings("blank"); got != nil {
		t.Errorf("ExtraStrings(blank) = %v, want nil", got)
	}
	if got := msg.ExtraStrings("nothing"); got != nil {
		t.Errorf("ExtraStrings(nothing) = %v, want nil", got)
	}
	if got := msg.ExtraStrings("absent"); got != nil {
		t.Errorf("ExtraStrings(absent) = %v, want nil", got)
	}
}

func TestExtraBool(t *testing.T) {
	msg := Message{Extras: map[string]any{"on": true, "off": false, "wrong": "yes"}}

	if v, ok := msg.ExtraBool("on"); !ok || !v {
		t.Errorf("ExtraBool(on) = %v,%v want true,true", v, ok)
	}
	// false must be distinguishable from absent: it changes what a channel sends.
	if v, ok := msg.ExtraBool("off"); !ok || v {
		t.Errorf("ExtraBool(off) = %v,%v want false,true", v, ok)
	}
	if _, ok := msg.ExtraBool("wrong"); ok {
		t.Errorf("ExtraBool(wrong type) ok = true, want false")
	}
	if _, ok := msg.ExtraBool("absent"); ok {
		t.Errorf("ExtraBool(absent) ok = true, want false")
	}
}

func TestExtraInt(t *testing.T) {
	msg := Message{Extras: map[string]any{
		"int":   7,
		"int32": int32(8),
		"int64": int64(9),
		// A number that came back from JSON is a float64.
		"json": float64(10),
		"str":  "11",
	}}

	for key, want := range map[string]int{"int": 7, "int32": 8, "int64": 9, "json": 10} {
		got, ok := msg.ExtraInt(key)
		if !ok || got != want {
			t.Errorf("ExtraInt(%s) = %v,%v want %v,true", key, got, ok, want)
		}
	}
	if _, ok := msg.ExtraInt("str"); ok {
		t.Errorf("ExtraInt(str) ok = true, want false")
	}
	if _, ok := msg.ExtraInt("absent"); ok {
		t.Errorf("ExtraInt(absent) ok = true, want false")
	}
}

func TestExtraOnNilMessage(t *testing.T) {
	var msg Message

	if msg.Extra("k") != nil || msg.ExtraString("k") != "" || msg.ExtraStrings("k") != nil {
		t.Errorf("reading Extras of a message without extras panicked or produced values")
	}
	if _, ok := msg.ExtraBool("k"); ok {
		t.Errorf("ExtraBool on nil extras ok = true, want false")
	}
	if _, ok := msg.ExtraInt("k"); ok {
		t.Errorf("ExtraInt on nil extras ok = true, want false")
	}
}
