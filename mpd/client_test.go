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

	// Add songs to the current playlist.
	files, err := cli.GetFiles()
	all := 4
	if err != nil {
		t.Errorf("Client.GetFiles failed: %s\n", err)
		return
	}
	if len(files) < all {
		t.Errorf("Add more then %d audio file to your MPD to run this test.", all)
		return
	}
	for i := 0; i < all; i++ {
		if err = cli.Add(files[i]); err != nil {
			t.Errorf("Client.Add failed: %s\n", err)
			return
		}
	}

	pls, err := cli.PlaylistInfo(-1, -1)
	if err != nil {
		// We can't use t.Fatalf because defer'ed calls won't run
		t.Errorf("Client.PlaylistInfo(-1, -1) = %v, %s need _, nil", pls, err)
		return
	}
	if len(pls) != all {
		t.Errorf("Client.PlaylistInfo(-1, -1) len = %d need %d", len(pls), all)
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

	pls, err = cli.PlaylistInfo(2, 4)
	if err != nil {
		t.Errorf("Client.PlaylistInfo(2, 4) = %v, %s need _, nil", pls, err)
		return
	}
	if len(pls) != 2 {
		t.Errorf("Client.PlaylistInfo(2, 4) len = %d need 2", len(pls))
		return
	}
}

func TestListInfo(t *testing.T) {
	cli := localDial(t)
	defer teardown(cli, t)

	fileCount, dirCount, plsCount := 0, 0, 0

	ls, err := cli.ListInfo("")
	if err != nil {
		// We can't use t.Fatalf because defer'ed calls won't run
		t.Errorf(`Client.ListInfo("") = %v, %s need _, nil`, ls, err)
		return
	}
	for i, item := range ls {
		if _, ok := item["file"]; ok {
			fileCount++
			for _, field := range []string{"last-modified", "artist", "title", "track"} {
				if _, ok := item[field]; !ok {
					t.Errorf(`ListInfo: file item %d has no "%s" field`, i, field)
				}
			}
		} else if _, ok := item["directory"]; ok {
			dirCount++
		} else if _, ok := item["playlist"]; ok {
			plsCount++
		} else {
			t.Errorf("ListInfo: item %d has no file/directory/playlist attribute", i)
		}
	}

	if expected := 100; fileCount != expected {
		t.Errorf(`ListInfo: expected %d files, got %d`, expected, fileCount)
	}
	if expected := 2; dirCount != expected {
		t.Errorf(`ListInfo: expected %d directories, got %d`, expected, dirCount)
	}
	if expected := 1; plsCount != expected {
		t.Errorf(`ListInfo: expected %d playlists, got %d`, expected, plsCount)
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

func TestListOutputs(t *testing.T) {
	cli := localDial(t)
	defer teardown(cli, t)

	outputs, err := cli.ListOutputs()
	if err != nil {
		t.Errorf(`Client.ListOutputs() = %v, %s need _, nil`, outputs, err)
		return
	}

	expected := []map[string]interface{}{}
	expected = append(expected,
		map[string]interface{}{"id": 0, "name": "downstairs", "enabled": true})
	expected = append(expected,
		map[string]interface{}{"id": 1, "name": "upstairs", "enabled": false})

	if len(outputs) != 2 {
		t.Errorf(`Listed %d outputs, expected %d`, len(outputs), 2)
	}
	for i, o := range outputs {
		if len(o) != 3 {
			t.Errorf(`Output should contain 3 keys, got %d`, len(o))
		}
		for k, v := range expected[i] {
			if outputs[i][k] != v {
				t.Errorf(`Expected property %o for key "%s", got %o`, v, k, outputs[i][k])
			}
		}
	}
}

func TestEnableOutput(t *testing.T) {
	cli := localDial(t)
	defer teardown(cli, t)

	err := cli.EnableOutput(1)
	if err != nil {
		t.Errorf("Client.EnableOutput failed: %s\n", err)
		return
	}
}

func TestDisableOutput(t *testing.T) {
	cli := localDial(t)
	defer teardown(cli, t)

	err := cli.DisableOutput(1)
	if err != nil {
		t.Errorf("Client.DisableOutput failed: %s\n", err)
		return
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
