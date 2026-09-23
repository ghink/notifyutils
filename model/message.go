package model

// Message is the one payload every channel understands. It deliberately holds
// only the intersection of what channels can express; anything channel-specific
// travels in Extras under a key the owning driver exports.
type Message struct {
	// Title is the short headline: email subject, notification title, markdown
	// heading. Channels without a title slot fold it into the body.
	Title string
	// Text is the body, interpreted according to Format.
	Text string
	// Format of Text. An empty Format is normalised to FormatPlain by the core
	// before dispatch, so drivers always see a concrete value.
	Format Format
	// Level selects the route in Config.Routes. An empty Level is normalised to
	// LevelInfo by the core before dispatch.
	Level Level

	// Recipients is per-message addressing, and its meaning is defined by each
	// driver: phone numbers for SMS, chat IDs for Telegram, mailbox addresses for
	// email, mobile numbers to mention for DingTalk and WeCom. When empty the
	// driver falls back to the default targets configured in its credentials.
	Recipients []string

	// Template is a provider-side template identifier, meaningful only to
	// channels that pre-register templates (SMS).
	Template string
	// Vars binds the template variables. Ordered, because some providers bind
	// positionally while others bind by key. The core renders them into Title and
	// Text before dispatch; channels that hand them to a provider read them here.
	Vars Vars

	// Extras carries channel-specific options. Keys are constants exported by
	// the driver that owns them; values keep their concrete Go types, so a
	// driver may document a struct or slice type under one of its keys.
	// The core forwards Extras verbatim and never reads it.
	Extras map[string]any
}

// Extra returns the raw value stored under key.
func (m Message) Extra(key string) any {
	return m.Extras[key]
}

// ExtraString returns the string stored under key, or "" when absent or not a string.
func (m Message) ExtraString(key string) string {
	v, _ := m.Extras[key].(string)
	return v
}

// ExtraStrings returns the strings stored under key, or nil when absent. A single
// string is accepted as one element, since callers often have only one value.
func (m Message) ExtraStrings(key string) []string {
	switch v := m.Extras[key].(type) {
	case []string:
		return v
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	default:
		return nil
	}
}

// ExtraBool returns the bool stored under key, and whether it was present.
func (m Message) ExtraBool(key string) (bool, bool) {
	v, ok := m.Extras[key].(bool)
	return v, ok
}

// ExtraInt returns the integer stored under key, and whether one was present.
// It accepts the whole numeric family so that a value that came back from JSON
// (float64) still reads as an int.
func (m Message) ExtraInt(key string) (int, bool) {
	switch v := m.Extras[key].(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}
