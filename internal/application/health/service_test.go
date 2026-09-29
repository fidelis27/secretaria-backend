package health

import (
	"context"
	"errors"
	"testing"
)

type checkerStub struct {
	err error
}

func (stub checkerStub) Check(context.Context) error {
	return stub.err
}

func TestServiceReportsDatabaseStatus(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		want        bool
		wantMessage string
	}{
		{name: "available", want: true, wantMessage: "database available"},
		{name: "unavailable", err: errors.New("ping MariaDB: connection refused"), want: false, wantMessage: "database unavailable"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status := NewService(checkerStub{err: test.err}).Check(context.Background())
			if status.OK != test.want {
				t.Fatalf("health status = %t, want %t", status.OK, test.want)
			}
			if status.Message != test.wantMessage {
				t.Fatalf("health message = %q, want %q", status.Message, test.wantMessage)
			}
		})
	}
}
