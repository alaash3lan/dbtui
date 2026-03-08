package statusbar

import "testing"

func TestNewStatusBar(t *testing.T) {
	m := New("testdb", "root", "127.0.0.1")

	if m.dbName != "testdb" {
		t.Errorf("dbName = %q, want %q", m.dbName, "testdb")
	}
	if m.user != "root" {
		t.Errorf("user = %q, want %q", m.user, "root")
	}
	if m.host != "127.0.0.1" {
		t.Errorf("host = %q, want %q", m.host, "127.0.0.1")
	}
	if m.connectionStatus != "" {
		t.Errorf("connectionStatus = %q, want empty", m.connectionStatus)
	}
	if m.queryTime != 0 {
		t.Errorf("queryTime = %v, want 0", m.queryTime)
	}
}

func TestSetConnectionStatus(t *testing.T) {
	m := New("testdb", "root", "127.0.0.1")

	m.SetConnectionStatus("reconnecting...")
	if m.connectionStatus != "reconnecting..." {
		t.Errorf("connectionStatus = %q, want %q", m.connectionStatus, "reconnecting...")
	}

	m.SetConnectionStatus("")
	if m.connectionStatus != "" {
		t.Errorf("connectionStatus = %q, want empty after clear", m.connectionStatus)
	}
}
