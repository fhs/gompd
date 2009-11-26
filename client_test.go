package client_test

import (
	. "mpd";
	"testing";
)

func localConnect(t *testing.T) (cli *Client) {
	addr := "127.0.0.1:6600";
	cli, err := Connect(addr);
	if err != nil {
		t.Fatalf("Connect(%q) = %v, %s want PTR, nil", addr, cli, err);
	}
	return;
}

func TestPlaylistInfo(t *testing.T) {
	cli := localConnect(t);
	defer cli.Disconnect();
	
	pls, err := cli.PlaylistInfo(-1, -1);
	if err != nil {
		// We can't use t.Fatalf because defer'ed calls won't run
		t.Errorf("Client.PlaylistInfo() = %v, %s need _, nil", pls, err);
		return;
	}
	for i, song := range pls {
		if _, ok := song["file"]; !ok {
			t.Errorf(`PlaylistInfo: song %d has no "file" attribute`, i);
		}
	}
}
