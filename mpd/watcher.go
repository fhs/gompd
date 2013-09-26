// Copyright 2009 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package mpd

import "io"

// Subsystem represents a subsystem that can be watched.
// See http://www.musicpd.org/doc/protocol/ch03.html#command_idle
// for valid subsystem names.
type Subsystem string

// Watcher represents a MPD client connection that can be watched for events.
type Watcher struct {
	conn  *Client        // client connection to MPD
	names chan []string  // channel to set new subsystems to watch
	Event chan Subsystem // event channel
	Error chan error     // error channel
}

// NewWatcher connects to MPD server and watches for changes in subsystems
// subsystems. If no subsystem is specified, all changes are reported.
func NewWatcher(net, addr, passwd string, names ...string) (w *Watcher, err error) {
	conn, err := DialAuthenticated(net, addr, passwd)
	if err != nil {
		return
	}
	w = &Watcher{
		conn: conn,
		// Buffered channel to avoid race conditions in Watcher.Subsystems().
		names: make(chan []string, 10),
		Event: make(chan Subsystem),
		Error: make(chan error),
	}
	go w.watch(names...)
	return
}

func (w *Watcher) watch(names ...string) {
	defer close(w.names)
	defer close(w.Event)
	defer close(w.Error)

	for {
		switch changed, err := w.conn.idle(names...); {
		case err == io.EOF:
			// Connection closed.
			w.Error <- err
			return
		case err != nil:
			w.Error <- err
		default:
			for _, sub := range changed {
				w.Event <- Subsystem(sub)
			}
		}

		select {
		case names = <-w.names:
			// Received new subsystems to watch.
		default:
			// continue
		}
	}
}

// Subsystems changes the subsystems to watch for.
func (w *Watcher) Subsystems(names ...string) {
	w.names <- names
	w.conn.noIdle()
}

// Close closes the connection to MPD and stops watching for events.
func (w *Watcher) Close() error {
	return w.conn.Close()
}
