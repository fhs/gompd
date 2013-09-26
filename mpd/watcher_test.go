package mpd

import (
	"testing"
	"time"
)

func localWatch(t *testing.T, names ...string) *Watcher {
	net, addr := localAddr()
	w, err := NewWatcher(net, addr, "", names...)
	if err != nil {
		t.Fatalf("NewWatcher(%q) = %v, %s want PTR, nil", addr, w, err)
	}
	return w
}

func TestWatcher(t *testing.T) {
	w := localWatch(t, "player")
	defer w.Close()

	c := localDial(t)
	defer teardown(c, t)

	// Give the watcher a chance.
	<-time.After(time.Second)

	// Trigger a player change.
	if err := c.Play(-1); err != nil {
		t.Errorf("Client.Play failed: %s\n", err)
		return
	}
	if err := c.Stop(); err != nil {
		t.Errorf("Client.Stop failed: %s\n", err)
		return
	}

	select {
	case subsystem := <-w.Event:
		if subsystem != "player" {
			t.Errorf("Unexpected result: %s != \"player\"\n", subsystem)
			return
		}
	case err := <-w.Error:
		t.Errorf("Client.idle failed: %s\n", err)
		return
	}

	w.Subsystems("update", "mixer")
	if err := c.Play(-1); err != nil {
		t.Errorf("Client.Play failed: %s\n", err)
		return
	}
	if _, err := c.Update(""); err != nil {
		t.Errorf("Client.Update failed: %s\n", err)
		return
	}

	select {
	case subsystem := <-w.Event:
		if subsystem != "update" {
			t.Errorf("Unexpected result: %s != \"update\"\n", subsystem)
			return
		}
	case err := <-w.Error:
		t.Errorf("Client.idle failed: %s\n", err)
		return
	}
}
