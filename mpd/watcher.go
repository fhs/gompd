// Copyright 2013 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package mpd

// Watcher represents a MPD client connection that can be watched for events.
type Watcher struct {
	conn  *Client       // client connection to MPD
	exit  chan bool     // channel used to ask loop to terminate
	done  chan bool     // channel indicating loop has terminated
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
		conn:  conn,
		Event: make(chan string),
		Error: make(chan error),
		done:  make(chan bool),
		// Buffer channels to avoid race conditions with noIdle
		names: make(chan []string, 1),
		exit:  make(chan bool, 1),
	}
	go w.watch(names...)
	return
}

func (w *Watcher) watch(names ...string) {
	defer w.closeChans()

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
		case <-w.exit:
			return
		case names = <-w.names:
			// Received new subsystems to watch.
		default:
			// continue
		}
	}
}

func (w *Watcher) closeChans() {
	close(w.Event)
	close(w.Error)
	close(w.names)
	close(w.exit)
	close(w.done)
}

// Subsystems changes the subsystems to watch for.
func (w *Watcher) Subsystems(names ...string) {
	w.names <- names
	w.conn.noIdle()
}

// Close closes the connection to MPD and stops watching for events.
func (w *Watcher) Close() error {
	w.exit <- true
	w.conn.noIdle()

	<-w.done // wait for idle to finish and channels to close
	// At this point, watch goroutine has ended,
	// so it's safe to close connection.
	return w.conn.Close()
}
