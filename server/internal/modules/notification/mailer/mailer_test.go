package mailer

import (
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"testing"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil is not a failure", nil, false},

		{"421 service closing", &textproto.Error{Code: 421, Msg: "closing transmission"}, true},
		{"450 mailbox busy", &textproto.Error{Code: 450, Msg: "mailbox unavailable"}, true},
		{"451 local error", &textproto.Error{Code: 451, Msg: "local error"}, true},
		{"452 insufficient storage", &textproto.Error{Code: 452, Msg: "out of storage"}, true},

		{"550 unknown mailbox", &textproto.Error{Code: 550, Msg: "no such user"}, false},
		{"553 bad address", &textproto.Error{Code: 553, Msg: "bad address"}, false},
		{"554 refused", &textproto.Error{Code: 554, Msg: "transaction failed"}, false},

		{"wrapped transient code", fmt.Errorf("smtp: %w", &textproto.Error{Code: 450}), true},
		{"wrapped permanent code", fmt.Errorf("smtp: %w", &textproto.Error{Code: 550}), false},

		{"malformed recipient is permanent", fmt.Errorf("%w: %q", ErrInvalidRecipient, "nope"), false},

		{"network timeout", &net.DNSError{IsTimeout: true}, true},

		{"unknown error retries", errors.New("connection reset by peer"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.want {
				t.Errorf("IsRetryable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestSendRejectsMalformedRecipientBeforeDialing(t *testing.T) {
	m := New(Config{Host: "127.0.0.1", Port: "1", From: "a@example.com"})

	err := m.Send(t.Context(), "not-an-address", "subject", "body")

	if !errors.Is(err, ErrInvalidRecipient) {
		t.Fatalf("err = %v, want ErrInvalidRecipient", err)
	}
	if IsRetryable(err) {
		t.Error("a malformed address must not be retried")
	}
}
