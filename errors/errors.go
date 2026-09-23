package errors

import "go.gh.ink/toolbox/pointer"

type NotifyutilsError struct {
	message string

	driverName      string
	driverCode      string
	driverMessage   string
	driverRequestID string
	driverResponse  any

	raw error
}

func (e *NotifyutilsError) Error() string {
	return e.message
}

func (e *NotifyutilsError) Is(err error) bool {
	return e.raw == err
}

// Unwrap reports no wrapped error.
//
// The raw field is only a matching anchor used by Is so that derived errors
// (produced by the With* copies) still match their origin sentinel. It is not
// a wrapped inner error, and in particular a sentinel's raw points at itself;
// returning it here would make errors.Is loop forever. Returning nil ends the
// unwrap chain while Is still provides sentinel matching.
func (e *NotifyutilsError) Unwrap() error {
	return nil
}

// clone creates a copy for chaining, preserving the reference to the original sentinel error.
// Internal helper used by all With* methods to ensure errors.Is always works.
func (e *NotifyutilsError) clone() *NotifyutilsError {
	ne := pointer.Copy(e)
	if ne.raw == nil {
		ne.raw = e
	}
	return ne
}

func (e *NotifyutilsError) WithDriverName(driverName string) *NotifyutilsError {
	ne := e.clone()
	ne.driverName = driverName
	return ne
}

func (e *NotifyutilsError) DriverName() string {
	return e.driverName
}

func (e *NotifyutilsError) WithDriverCode(code string) *NotifyutilsError {
	ne := e.clone()
	ne.driverCode = code
	return ne
}

func (e *NotifyutilsError) DriverCode() string {
	return e.driverCode
}

func (e *NotifyutilsError) WithDriverMessage(message string) *NotifyutilsError {
	ne := e.clone()
	ne.driverMessage = message
	return ne
}

func (e *NotifyutilsError) DriverMessage() string {
	return e.driverMessage
}

func (e *NotifyutilsError) WithDriverRequestID(requestID string) *NotifyutilsError {
	ne := e.clone()
	ne.driverRequestID = requestID
	return ne
}

func (e *NotifyutilsError) DriverRequestID() string {
	return e.driverRequestID
}

func (e *NotifyutilsError) WithDriverResponse(driverResponse any) *NotifyutilsError {
	ne := e.clone()
	ne.driverResponse = driverResponse
	return ne
}

func (e *NotifyutilsError) DriverResponse() any {
	return e.driverResponse
}

type Option func(*NotifyutilsError)

func WithDriverName(driverName string) Option {
	return func(e *NotifyutilsError) {
		e.driverName = driverName
	}
}

func WithDriverCode(driverCode string) Option {
	return func(e *NotifyutilsError) {
		e.driverCode = driverCode
	}
}

func WithDriverMessage(driverMessage string) Option {
	return func(e *NotifyutilsError) {
		e.driverMessage = driverMessage
	}
}

func WithDriverRequestID(driverRequestID string) Option {
	return func(e *NotifyutilsError) {
		e.driverRequestID = driverRequestID
	}
}

func WithDriverResponse(driverResponse any) Option {
	return func(e *NotifyutilsError) {
		e.driverResponse = driverResponse
	}
}

func New(c string, options ...Option) *NotifyutilsError {
	err := &NotifyutilsError{message: c}

	for _, option := range options {
		option(err)
	}

	err.raw = err

	return err
}

// Config

var ErrDriverNotRegistered = New("driver not registered")
var ErrDriverCredentialInvalid = New("driver credential invalid")

// Routing

var ErrDriverNotFound = New("driver not found")
var ErrNoDriverConfigured = New("no driver configured")

// Sending

var ErrDriverSendFailed = New("driver send failed")
var ErrRecipientRequired = New("no recipient: the message carries none and the credentials define no default")
var ErrTemplateRequired = New("template required: this channel delivers only pre-registered templates")
var ErrUnsupportedFormat = New("unsupported message format")
var ErrMessageTooLong = New("message too long")
