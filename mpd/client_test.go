// Copyright 2009 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package mpd

import (
	"os"
	"testing"
)

func localDial(t *testing.T) (cli *Client) {
	net := "unix"
	addr := os.Getenv("MPD_HOST")
	if addr == "" {
		addr = "localhost"
	}
	if addr[0] != '/' {
		net = "tcp"
		port := os.Getenv("MPD_PORT")
		if port == "" {
			port = "6600"
		}
		addr += ":" + port
	}
	cli, err := Dial(net, addr)
	if err != nil {
		t.Fatalf("Dial(%q) = %v, %s want PTR, nil", addr, cli, err)
	}
	return
}

func close(cli *Client, t *testing.T) {
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
	defer close(cli, t)

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
			t.Errorf("Inconsistent song attribute for song %d", i)
		}
	}
}

func TestCurrentSong(t *testing.T) {
	cli := localDial(t)
	defer close(cli, t)

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
	defer close(cli, t)

	err := cli.Ping()
	if err != nil {
		t.Errorf("Client.Ping failed: %s\n", err)
	}
}

func TestUpdate(t *testing.T) {
	cli := localDial(t)
	defer close(cli, t)

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
	defer close(cli, t)

	files, err := cli.GetFiles()
	if err != nil {
		t.Errorf("Client.GetFiles failed: %s\n", err)
		return
	}
	if len(files) < 2 {
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
	found := false
	for _, pl := range attrs {
		if pl["playlist"] == "Test Playlist" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Couldn't find playlist \"Test Playlist\" in %v\n", attrs)
		return
	}
	attrs, err = cli.ListPlaylistInfo("Test Playlist")
	if err != nil {
		t.Errorf("Client.PlaylistInfo failed: %s\n", err)
		return
	}
	found = false
	for _, song := range attrs {
		if song["file"] == files[0] {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Couldn't find song \"%s\" in %v", attrs)
		return
	}
	if err = cli.PlaylistDelete("Test Playlist", 0); err != nil {
		t.Errorf("Client.PlaylistDelete failed: %s\n", err)
		return
	}
	playlist, err := cli.ListPlaylistInfo("Test Playlist")
	if err != nil {
		t.Errorf("Client.ListPlaylistInfo failed: %s\n", err)
		return
	}
	if len(playlist) != len(attrs)-1 {
		t.Errorf("Unxpected number of tracks in the playlist: %d != %d", len(playlist), len(attrs))
		return
	}
	if err = cli.Rename("Test Playlist", "Test Playlist 2"); err != nil {
		t.Errorf("Client.Rename failed: %s\n", err)
		return
	}
	if err = cli.Clear(); err != nil {
		t.Errorf("Client.Clear failed: %s\n", err)
		return
	}
	if err = cli.Load("Test Playlist 2", -1, -1); err != nil {
		t.Errorf("Client.Load failed: %s\n", err)
		return
	}
	attrs, err = cli.PlaylistInfo(-1, -1)
	if err != nil {
		t.Errorf("Client.PlaylistInfo failed: %s\n", err)
		return
	}
	for i, attr := range attrs {
		if attr["file"] != playlist[i]["file"] {
			t.Errorf("Unexpected file: %s != %s\n", attr["files"], playlist[i]["file"])
			return
		}
	}
	if err = cli.PlaylistClear("Test Playlist 2"); err != nil {
		t.Errorf("Client.Clear failed: %s\n", err)
		return
	}
	attrs, err = cli.ListPlaylistInfo("Test Playlist 2")
	if err != nil {
		t.Errorf("Client.ListPlaylistInfo failed: %s\n", err)
		return
	}
	if len(attrs) != 0 {
		t.Errorf("Unexpected number of songs: %d != 0\n", len(attrs))
		return
	}
	if err = cli.Rm("Test Playlist 2"); err != nil {
		t.Errorf("Client.Rm failed: %s\n", err)
		return
	}
	attrs, err = cli.ListPlaylists()
	if err != nil {
		t.Errorf("Client.ListPlaylists failed: %s\n", err)
		return
	}
	for _, attr := range attrs {
		if attr["playlist"] == "Test Playlist 2" {
			t.Errorf("Found playlist \"Test Playlist 2\" in %v\n", attrs)
			return
		}
	}
	if err = cli.Save("Test Playlist"); err != nil {
		t.Errorf("Client.Save failed: %s\n", err)
		return
	}
	attrs, err = cli.ListPlaylistInfo("Test Playlist")
	if err != nil {
		t.Errorf("Client.ListPlaylistInfo failed: %s\n", err)
		return
	}
	for i, attr := range attrs {
		if attr["file"] != playlist[i]["file"] {
			t.Errorf("Unexpected file: %s != %s\n", attr["files"], playlist[i]["file"])
			return
		}
	}
}
