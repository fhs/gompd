// Copyright © 2009 Fazlul Shahriar <fshahriar@gmail.com>.
// See LICENSE file for license details.

package client_test

import (
	. "mpd";
	"testing";
)

func localConnect(t *testing.T) (cli *Client) {
	addr := "127.0.0.1:6600";
	cli, err := Connect("tcp", addr);
	if err != nil {
		t.Fatalf("Connect(%q) = %v, %s want PTR, nil", addr, cli, err)
	}
	return;
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
	return true;
}

func TestPlaylistInfo(t *testing.T) {
	cli := localConnect(t);
	defer cli.Close();

	pls, err := cli.PlaylistInfo(-1, -1);
	if err != nil {
		// We can't use t.Fatalf because defer'ed calls won't run
		t.Errorf("Client.PlaylistInfo(-1, -1) = %v, %s need _, nil", pls, err);
		return;
	}
	for i, song := range pls {
		if _, ok := song["file"]; !ok {
			t.Errorf(`PlaylistInfo: song %d has no "file" attribute`, i)
		}
		pls1, err := cli.PlaylistInfo(i, -1);
		if err != nil {
			t.Errorf("Client.PlaylistInfo(%d, -1) = %v, %s need _, nil", pls1, err)
		}
		if !attrsEqual(pls[i], pls1[0]) {
			t.Errorf("Inconsistent song attribute for song %d", i)
		}
	}
}
