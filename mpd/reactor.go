// Copyright 2013 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package mpd

type Reactor struct {
	c       *Client       // client connection to MPD
	exit    chan struct{} // channel used to ask loop to terminate
	intr    chan interface{}
	handler Handler
}

type Handler func(c *Client, intr interface{}, subsystems []string, err error)

func (r *Reactor) closeChans() {
	close(r.exit)
	close(r.intr)
}

func NewReactor(
	net, addr, passwd string,
	handler Handler,
	subsystems ...string,
) (w *Reactor, err error) {
	c, err := DialAuthenticated(net, addr, passwd)
	if err != nil {
		return nil, err
	}

	r := &Reactor{
		c:       c,
		handler: handler,
		exit:    make(chan struct{}),
	}

	go r.watch(subsystems...)
	return r, nil
}

func (r *Reactor) watch(subsystems ...string) {
	defer r.closeChans()
	for {
		intr := interface{}(nil)
		changed, err := r.c.idle(subsystems...)
		select {
		case <-r.exit:
			return
		case intr = <-r.intr:
		default:
		}
		r.handler(r.c, intr, changed, err)
	}
}

func (r *Reactor) Interrupt(arg interface{}) error {
	err := r.c.noIdle()
	if err != nil {
		return err
	}
	r.intr <- arg
	return nil
}

func (r *Reactor) Close() error {
	err := r.c.noIdle()
	if err != nil {
		return err
	}
	r.exit <- struct{}{}
	<-r.exit
	return r.c.Close()
}
