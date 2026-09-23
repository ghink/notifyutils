package model

// Format tells a driver which body flavour the message text is written in.
// A driver that cannot render the requested format must fail with
// ErrUnsupportedFormat rather than silently reinterpreting the text.
type Format = string

const (
	// FormatPlain is literal text; markup characters carry no meaning.
	FormatPlain Format = "plain"
	// FormatMarkdown is CommonMark-ish text; each channel applies its own dialect.
	FormatMarkdown Format = "markdown"
	// FormatHTML is an HTML fragment, honoured only by channels that render it (email).
	FormatHTML Format = "html"
)
