// Copyright 2013 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package mpd

// Watcher represents a MPD client connection that can be watched for events.
type Watcher struct {
	conn  *Client       // client connection to MPD
	done  chan bool     // channel to stop the loop
	names chan []string // channel to set new subsystems to watch
	Event chan string   // event channel
	Error chan error    // error channel
}

// NewWatcher connects to MPD server and watches for changes in subsystems
// names. If no subsystem is specified, all changes are reported.
//
// See http://www.musicpd.org/doc/protocol/ch03.html#command_idle for valid
// subsystem names.
func NewWatcher(net, addr, passwd string, names ...string) (w *Watcher, err error) {
	conn, err := DialAuthenticated(net, addr, passwd)
	if err != nil {
		return
	}
	w = &Watcher{
		conn: conn,
		done: make(chan bool),
		// Buffered channel to avoid race conditions in Watcher.Subsystems().
		names: make(chan []string, 1),
		Event: make(chan string, 100),
		Error: make(chan error, 100),
	}
	go w.watch(names...)
	return
}

func (w *Watcher) watch(names ...string) {
	defer w.close()

	for {
		switch changed, err := w.conn.idle(names...); {
		case err != nil:
			w.Error <- err
		default:
			for _, name := range changed {
				w.Event <- name
			}
		}

		select {
		case <-w.done:
			return
		case names = <-w.names:
			// Received new subsystems to watch.
		default:
			// continue
		}
	}
}

func (w *Watcher) close() {
	close(w.done)
	close(w.names)
	close(w.Event)
	close(w.Error)
	w.conn.Close()
}

// Subsystems changes the subsystems to watch for.
func (w *Watcher) Subsystems(names ...string) {
	w.names <- names
	w.conn.noIdle()
}

// Close closes the connection to MPD and stops watching for events.
func (w *Watcher) Close() {
	w.conn.noIdle() // This is a little bit racy.
	w.done <- true  // Let's see, if it's a problem in practice.
}
