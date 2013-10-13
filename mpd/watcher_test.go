// Copyright 2013 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

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

func loadTestFiles(t *testing.T, cli *Client, n int) (ok bool) {
	if err := cli.Clear(); err != nil {
		t.Errorf("Client.Clear failed: %s\n", err)
		return
	}
	files, err := cli.GetFiles()
	if err != nil {
		t.Errorf("Client.GetFiles failed: %s\n", err)
		return
	}
	if len(files) < n {
		t.Log("Add files to your MPD to run this test.")
		return
	}
	for i := 0; i < n; i++ {
		if err = cli.Add(files[i]); err != nil {
			t.Errorf("Client.Add failed: %s\n", err)
			return
		}
	}
	return true
}

func TestWatcher(t *testing.T) {
	w := localWatch(t, "player")
	defer w.Close()

	c := localDial(t)
	defer teardown(c, t)
	if !loadTestFiles(t, c, 10) {
		return
	}

	// Give the watcher a chance.
	<-time.After(time.Second)

	if err := c.Play(-1); err != nil { // player change
		t.Errorf("Client.Play failed: %s\n", err)
		return
	}
	if err := c.Next(); err != nil { // player change
		t.Errorf("Client.Next failed: %s\n", err)
		return
	}
	if err := c.Previous(); err != nil { // player change
		t.Errorf("Client.Previous failed: %s\n", err)
		return
	}

	select {
	case subsystem := <-w.Event:
		if subsystem != "player" {
			t.Errorf("Unexpected result: %q != \"player\"\n", subsystem)
			return
		}
	case err := <-w.Error:
		t.Errorf("Client.idle failed: %s\n", err)
		return
	}

	w.Subsystems("options", "playlist")
	if err := c.Stop(); err != nil { // player change
		t.Errorf("Client.Stop failed: %s\n", err)
		return
	}
	if err := c.Delete(5, -1); err != nil { // playlist change
		t.Errorf("Client.Delete failed: %s\n", err)
		return
	}

	select {
	case subsystem := <-w.Event:
		if subsystem != "playlist" {
			t.Errorf("Unexpected result: %q != \"playlist\"\n", subsystem)
			return
		}
	case err := <-w.Error:
		t.Errorf("Client.idle failed: %s\n", err)
		return
	}
}
