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
		t.Fatalf("Client.Clear failed: %s\n", err)
	}
	files, err := cli.GetFiles()
	if err != nil {
		t.Fatalf("Client.GetFiles failed: %s\n", err)
	}
	if len(files) < n {
		t.Log("Add files to your MPD to run this test.")
		return
	}
	for i := 0; i < n; i++ {
		if err = cli.Add(files[i]); err != nil {
			t.Fatalf("Client.Add failed: %s\n", err)
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
		t.Fatalf("Client.Play failed: %s\n", err)
	}
	if err := c.Next(); err != nil { // player change
		t.Fatalf("Client.Next failed: %s\n", err)
	}
	if err := c.Previous(); err != nil { // player change
		t.Fatalf("Client.Previous failed: %s\n", err)
	}

	select {
	case subsystem := <-w.Event:
		if subsystem != "player" {
			t.Fatalf("Unexpected result: %q != \"player\"\n", subsystem)
		}
	case err := <-w.Error:
		t.Fatalf("Client.idle failed: %s\n", err)
	}

	w.Subsystems("options", "playlist")
	if err := c.Stop(); err != nil { // player change
		t.Fatalf("Client.Stop failed: %s\n", err)
	}
	if err := c.Delete(5, -1); err != nil { // playlist change
		t.Fatalf("Client.Delete failed: %s\n", err)
	}

	select {
	case subsystem := <-w.Event:
		if subsystem != "playlist" {
			t.Fatalf("Unexpected result: %q != \"playlist\"\n", subsystem)
		}
	case err := <-w.Error:
		t.Fatalf("Client.idle failed: %s\n", err)
	}
}
