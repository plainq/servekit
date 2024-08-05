package litekit

import (
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestConn_connString(t *testing.T) {
	tests := map[string]struct {
		conn Conn

		want    string
		wantErr error
	}{
		"PathOnly": {
			conn:    Conn{path: "/path/to/db", accessMode: ReadWriteCreate, journalingMode: Delete},
			want:    "file:/path/to/db?mode=rwc&_journal=DELETE",
			wantErr: nil,
		},

		"ReadWrite": {
			conn:    Conn{path: "/path/to/db", accessMode: ReadWrite, journalingMode: Delete},
			want:    "file:/path/to/db?mode=rw&_journal=DELETE",
			wantErr: nil,
		},

		"ReadOnly": {
			conn:    Conn{path: "/path/to/db", accessMode: ReadOnly, journalingMode: Delete},
			want:    "file:/path/to/db?mode=ro&_journal=DELETE",
			wantErr: nil,
		},

		"InMemory": {
			conn:    Conn{path: "/path/to/db", accessMode: InMemory, journalingMode: Delete},
			want:    "file:/path/to/db?mode=memory&_journal=DELETE",
			wantErr: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tc.conn.connString()
			td.Cmp(t, err, tc.wantErr)
			td.Cmp(t, got, tc.want)
		})
	}
}
