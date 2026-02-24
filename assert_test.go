package sqlhelper

import "testing"

func testSQL(t *testing.T, name string, sql string, expected string) {
	if sql != expected {
		t.Errorf("%s:\n  got:  %s\n  want: %s", name, sql, expected)
	}
}

func testArgsLen(t *testing.T, name string, args []interface{}, expected int) {
	if len(args) != expected {
		t.Errorf("%s: Args length mismatch: got %d, want %d", name, len(args), expected)
	}
}
