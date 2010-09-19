// Copyright © 2010 Fazlul Shahriar <fshahriar@gmail.com>.
// See LICENSE file for license details.

package main

import (
	"log"
	"gompd.googlecode.com/hg/mpd"
	"goplan9.googlecode.com/hg/plan9/acme"
)

func main() {
	w, err := acme.New()
	if err != nil {
		log.Exit(err)
	}
	w.Name("/ampc/")

	cli, err := mpd.Dial("tcp", "localhost:6600")
	if err != nil {
		log.Exit(err)
	}
	defer cli.Close()

	pls, err := cli.PlaylistInfo(-1, -1)
	if err != nil {
		log.Exit(err)
	}

	for _, song := range pls {
		w.Printf("body", "%s: %s\n", song["Pos"], song["file"])
	}
	w.Ctl("clean")

	for e := range w.EventChan() {
		switch e.C2 {
		case 'x', 'X': // execute
			if string(e.Text) == "Del" {
				w.Ctl("delete")
			}
			w.WriteEvent(e)
		}
	}
	w.CloseFiles()
}
