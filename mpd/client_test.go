// Copyright 2009 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package mpd

import (
	"os"
	"testing"
)

var (
	serverRunning  = false
	useGoMPDServer = true
)

func localAddr() (net, addr string) {
	if useGoMPDServer {
		// Don't clash with standard MPD port 6600
		return "tcp", "127.0.0.1:6603"
	}
	net = "unix"
	addr = os.Getenv("MPD_HOST")
	if len(addr) > 0 && addr[0] == '/' {
		return
	}
	net = "tcp"
	if len(addr) == 0 {
		addr = "127.0.0.1"
	}
	port := os.Getenv("MPD_PORT")
	if len(port) == 0 {
		port = "6600"
	}
	return net, addr + ":" + port
}

func localDial(t *testing.T) *Client {
	net, addr := localAddr()
	if useGoMPDServer && !serverRunning {
		running := make(chan bool)
		go serve(net, addr, running)
		serverRunning = true
		<-running
	}
	cli, err := Dial(net, addr)
	if err != nil {
		t.Fatalf("Dial(%q) = %v, %s want PTR, nil", addr, cli, err)
	}
	return cli
}

func teardown(cli *Client, t *testing.T) {
	if err := cli.Close(); err != nil {
		t.Errorf("Client.Close() = %s need nil", err)
	}
}

func attrsEqual(left, right Attrs) bool {
	if len(left) != len(right) {
		return false
	}
	for key, lval := range left {
		if rval, ok := right[key]; !ok || lval != rval {
			return false
		}
	}
	return true
}

func TestPlaylistInfo(t *testing.T) {
	cli := localDial(t)
	defer teardown(cli, t)

	pls, err := cli.PlaylistInfo(-1, -1)
	if err != nil {
		// We can't use t.Fatalf because defer'ed calls won't run
		t.Errorf("Client.PlaylistInfo(-1, -1) = %v, %s need _, nil", pls, err)
		return
	}
	for i, song := range pls {
		if _, ok := song["file"]; !ok {
			t.Errorf(`PlaylistInfo: song %d has no "file" attribute`, i)
		}
		pls1, err := cli.PlaylistInfo(i, -1)
		if err != nil {
			t.Errorf("Client.PlaylistInfo(%d, -1) = %v, %s need _, nil", i, pls1, err)
		}
		if !attrsEqual(pls[i], pls1[0]) {
			t.Errorf("song at position %d is %v; want %v", i, pls[i], pls1[0])
		}
	}
}

func TestCurrentSong(t *testing.T) {
	cli := localDial(t)
	defer teardown(cli, t)

	attrs, err := cli.CurrentSong()
	if err != nil {
		t.Errorf("Client.CurrentSong() = %v, %s need _, nil", attrs, err)
		return
	}
	if len(attrs) == 0 {
		return // no current song
	}
	_, ok := attrs["file"]
	if !ok {
		t.Errorf("current song (attrs=%v) has no file attribute", attrs)
		return
	}
}

func TestPing(t *testing.T) {
	cli := localDial(t)
	defer teardown(cli, t)

	err := cli.Ping()
	if err != nil {
		t.Errorf("Client.Ping failed: %s\n", err)
	}
}

func TestUpdate(t *testing.T) {
	cli := localDial(t)
	defer teardown(cli, t)

	id, err := cli.Update("")
	if err != nil {
		t.Errorf("Client.Update failed: %s\n", err)
		return
	}
	if id < 1 {
		t.Errorf("job id is too small: %d\n", id)
	}
}

func TestPlaylistFunctions(t *testing.T) {
	cli := localDial(t)
	defer teardown(cli, t)

	files, err := cli.GetFiles()
	if err != nil {
		t.Errorf("Client.GetFiles failed: %s\n", err)
		return
	}
	if len(files) < 2 {
		t.Log("Add more then 1 audio file to your MPD to run this test.")
		return
	}
	for i := 0; i < 2; i++ {
		if err = cli.PlaylistAdd("Test Playlist", files[i]); err != nil {
			t.Errorf("Client.PlaylistAdd failed: %s\n", err)
			return
		}
	}
	attrs, err := cli.ListPlaylists()
	if err != nil {
		t.Errorf("Client.ListPlaylists failed: %s\n", err)
		return
	}
	if i := attrsListIndex(attrs, "playlist", "Test Playlist"); i < 0 {
		t.Errorf("Couldn't find playlist \"Test Playlist\" in %v\n", attrs)
		return
	}
	attrs, err = cli.PlaylistContents("Test Playlist")
	if err != nil {
		t.Errorf("Client.PlaylistContents failed: %s\n", err)
		return
	}
	if i := attrsListIndex(attrs, "file", files[0]); i < 0 {
		t.Errorf("Couldn't find song %q in %v", files[0], attrs)
		return
	}
	if err = cli.PlaylistDelete("Test Playlist", 0); err != nil {
		t.Errorf("Client.PlaylistDelete failed: %s\n", err)
		return
	}
	playlist, err := cli.PlaylistContents("Test Playlist")
	if err != nil {
		t.Errorf("Client.PlaylistContents failed: %s\n", err)
		return
	}
	if !attrsListEqual(playlist, attrs[1:]) {
		t.Errorf("PlaylistContents returned %v; want %v", playlist, attrs[1:])
		return
	}
	cli.PlaylistRemove("Test Playlist 2")
	if err = cli.PlaylistRename("Test Playlist", "Test Playlist 2"); err != nil {
		t.Errorf("Client.PlaylistRename failed: %s\n", err)
		return
	}
	if err = cli.Clear(); err != nil {
		t.Errorf("Client.Clear failed: %s\n", err)
		return
	}
	if err = cli.PlaylistLoad("Test Playlist 2", -1, -1); err != nil {
		t.Errorf("Client.Load failed: %s\n", err)
		return
	}
	attrs, err = cli.PlaylistInfo(-1, -1)
	if err != nil {
		t.Errorf("Client.PlaylistInfo failed: %s\n", err)
		return
	}
	if !attrsListEqualKey(playlist, attrs, "file") {
		t.Errorf("Unexpected playlist: %v != %v\n", attrs, playlist)
		return
	}
	if err = cli.PlaylistClear("Test Playlist 2"); err != nil {
		t.Errorf("Client.PlaylistClear failed: %s\n", err)
		return
	}
	attrs, err = cli.PlaylistContents("Test Playlist 2")
	if err != nil {
		t.Errorf("Client.PlaylistContents failed: %s\n", err)
		return
	}
	if len(attrs) != 0 {
		t.Errorf("Unexpected number of songs: %d != 0\n", len(attrs))
		return
	}
	if err = cli.PlaylistRemove("Test Playlist 2"); err != nil {
		t.Errorf("Client.PlaylistRemove failed: %s\n", err)
		return
	}
	attrs, err = cli.ListPlaylists()
	if err != nil {
		t.Errorf("Client.ListPlaylists failed: %s\n", err)
		return
	}
	if i := attrsListIndex(attrs, "playlist", "Test Playlist 2"); i > -1 {
		t.Errorf("Found playlist \"Test Playlist 2\" in %v\n", attrs)
		return
	}
	if err = cli.PlaylistSave("Test Playlist"); err != nil {
		t.Errorf("Client.PlaylistSave failed: %s\n", err)
		return
	}
	attrs, err = cli.PlaylistContents("Test Playlist")
	if err != nil {
		t.Errorf("Client.PlaylistContents failed: %s\n", err)
		return
	}
	if !attrsListEqual(playlist, attrs) {
		t.Errorf("Unexpected playlist: %v != %v\n", attrs, playlist)
		return
	}
}

func attrsListIndex(attrs []Attrs, key, value string) int {
	for i, attr := range attrs {
		if attr[key] == value {
			return i
		}
	}
	return -1
}

func attrsListEqual(a, b []Attrs) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !attrsEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func attrsListEqualKey(a, b []Attrs, key string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i][key] != b[i][key] {
			return false
		}
	}
	return true
}

var quoteTests = []struct {
	s, q string
}{
	{`test.ogg`, `"test.ogg"`},
	{`test "song".ogg`, `"test \"song\".ogg"`},
	{`04 - ILL - DECAYED LOVE　feat.℃iel.ogg`, `"04 - ILL - DECAYED LOVE　feat.℃iel.ogg"`},
}

func TestQuote(t *testing.T) {
	for _, test := range quoteTests {
		if q := quote(test.s); q != test.q {
			t.Errorf("quote(%s) returned %s; expected %s\n", test.s, q, test.q)
		}
	}
}
