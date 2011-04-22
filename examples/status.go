// Copyright 2009 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"gompd.googlecode.com/hg/mpd"
	"os"
	"time"
)

type Song struct {
	title  string
	artist string
	album  string
}

func currentSong(cli *mpd.Client) (song *Song) {
	song = new(Song)
	sattr, err := cli.CurrentSong()
	if err != nil {
		return
	}
	song.title, _ = sattr["Title"]
	song.artist, _ = sattr["Artist"]
	song.album, _ = sattr["Album"]
	return
}

func main() {
	//mpd.Chatty = true;
	cli, err := mpd.Dial("tcp", "127.0.0.1:6600")
	if err != nil {
		goto err
	}
	defer cli.Close()

	line := ""
	line1 := ""
	for {
		status, err := cli.Status()
		if err != nil {
			goto err
		}
		song := currentSong(cli)
		if status["state"] == "play" {
			line1 = fmt.Sprintf("%s - %s", song.artist, song.title)
		} else {
			line1 = fmt.Sprintf("State: %s", status["state"])
		}
		if line != line1 {
			line = line1
			fmt.Println(line)
		}
		time.Sleep(1e9)
	}
	return
err:
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
