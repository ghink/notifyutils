package text

import (
	"strings"
	"testing"

	"go.gh.ink/notifyutils/model"
)

func TestSubst(t *testing.T) {
	vars := model.Vars{
		{Key: "code", Value: "1234"},
		{Key: "name", Value: "Ada"},
	}

	if got := Subst("Your code is ${code}, ${name}", vars); got != "Your code is 1234, Ada" {
		t.Errorf("Subst() = %q", got)
	}
	// Repeated and out-of-order placeholders both bind.
	if got := Subst("${code}/${code}/${name}/${code}", vars); got != "1234/1234/Ada/1234" {
		t.Errorf("Subst(repeat) = %q", got)
	}
	// An unknown placeholder stays visible rather than vanishing.
	if got := Subst("hi ${missing}", vars); got != "hi ${missing}" {
		t.Errorf("Subst(unknown) = %q, want the placeholder kept", got)
	}
	if got := Subst("plain", nil); got != "plain" {
		t.Errorf("Subst(nil vars) = %q", got)
	}
}

func TestSubstSkipsNilVar(t *testing.T) {
	vars := model.Vars{nil, {Key: "a", Value: "1"}, nil}

	if got := Subst("${a}${b}", vars); got != "1${b}" {
		t.Errorf("Subst() = %q, want 1${b}", got)
	}
}

func TestChunkShortInput(t *testing.T) {
	in := "hello"
	got := Chunk(in, 10)
	if len(got) != 1 || got[0] != in {
		t.Errorf("Chunk() = %q, want the input unchanged", got)
	}
}

func TestChunkNonPositiveMax(t *testing.T) {
	// A bad limit must not loop forever; the whole message goes out as one piece.
	got := Chunk("hello world", 0)
	if len(got) != 1 || got[0] != "hello world" {
		t.Errorf("Chunk(max=0) = %q, want one chunk", got)
	}
}

func TestChunkKeepsEveryCharacter(t *testing.T) {
	in := strings.Repeat("abcdefghij", 50) // 500 runes, no break opportunity
	got := Chunk(in, 21)

	if strings.Join(got, "") != in {
		t.Errorf("Chunk() lost or reordered characters")
	}
	for i, c := range got {
		if Runes(c) > 21 {
			t.Errorf("chunk %d has %d runes, over the 21 limit", i, Runes(c))
		}
	}
}

func TestChunkBreaksOnTheLastWhitespaceThatFits(t *testing.T) {
	in := "alpha\nbravo charlie\ndelta"
	got := Chunk(in, 18)

	if len(got) != 2 {
		t.Fatalf("Chunk() = %q, want two pieces", got)
	}
	if got[0] != "alpha\nbravo" {
		t.Errorf("first chunk = %q, want the break at the last whitespace in the window", got[0])
	}
	if got[1] != "charlie\ndelta" {
		t.Errorf("second chunk = %q", got[1])
	}

	in = "alpha bravo charlie delta"
	got = Chunk(in, 12)
	if got[0] != "alpha bravo" {
		t.Errorf("first chunk = %q, want the break at the last space", got[0])
	}
}

func TestChunkHandlesMultiByteRunes(t *testing.T) {
	// Limits are stated in characters, so a CJK message must not be cut mid-rune.
	in := "服务不可用，请检查数据库连接池配置" + strings.Repeat("字", 30)
	got := Chunk(in, 10)

	if strings.Join(got, "") != in {
		t.Errorf("Chunk() lost characters: %q", got)
	}
	for _, c := range got {
		if Runes(c) > 10 {
			t.Errorf("chunk %q exceeds the limit", c)
		}
	}
}

func TestRunes(t *testing.T) {
	if got := Runes("中文abc"); got != 5 {
		t.Errorf("Runes() = %d, want 5", got)
	}
}
