// Copyright 2009 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package mpd_test

import (
	"fmt"
	"log"
	"time"

	"github.com/fhs/gompd/v2/mpd"
)

func ExampleDial() {
	// Connect to MPD server
	conn, err := mpd.Dial("tcp", "localhost:6600")
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	line := ""
	line1 := ""
	// Loop printing the current status of MPD.
	for {
		status, err := conn.Status()
		if err != nil {
			log.Fatalln(err)
		}
		song, err := conn.CurrentSong()
		if err != nil {
			log.Fatalln(err)
		}
		if status["state"] == "play" {
			line1 = fmt.Sprintf("%s - %s", song["Artist"], song["Title"])
		} else {
			line1 = fmt.Sprintf("State: %s", status["state"])
		}
		if line != line1 {
			line = line1
			fmt.Println(line)
		}
		time.Sleep(1e9)
	}
}

func ExampleNewWatcher() {
	w, err := mpd.NewWatcher("tcp", ":6600", "")
	if err != nil {
		log.Fatalln(err)
	}
	defer w.Close()

	// Log errors.
	go func() {
		for err := range w.Error {
			log.Println("Error:", err)
		}
	}()

	// Log events.
	go func() {
		for subsystem := range w.Event {
			log.Println("Changed subsystem:", subsystem)
		}
	}()

	// Do other stuff...
	time.Sleep(3 * time.Minute)
}
