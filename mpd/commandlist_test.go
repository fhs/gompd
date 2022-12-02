// Copyright 2013 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package mpd

import (
	"testing"
)

func TestCurrentSongPromise(t *testing.T) {
	cli := localDial(t)
	defer teardown(cli, t)

	cmdl := cli.BeginCommandList()

	pa := cmdl.CurrentSong()

	if err := cmdl.End(); err != nil {
		t.Errorf("CommandList.End failed: %s", err)
	}

	if _, err := pa.Value(); err != nil {
		t.Errorf("Promise did not compute: %s", err)
	}

}

func TestCommandList(t *testing.T) {
	cli := localDial(t)
	defer teardown(cli, t)

	// Normal command list:
	cmdl := cli.BeginCommandList()

	cmdl.Next()
	cmdl.Next()
	cmdl.Next()

	if err := cmdl.End(); err != nil {
		t.Errorf("CommandList.End failed: %s", err)
	}

	// Test empty command list:
	cmdl2 := cli.BeginCommandList()
	if err := cmdl2.End(); err != nil {
		t.Errorf("CommandList.End failed: %s", err)
	}

	// Reuse old comandlist (should work):
	cmdl.Previous()
	cmdl.Previous()
	cmdl.Previous()
	if err := cmdl.End(); err != nil {
		t.Errorf("CommandList.End failed: %s", err)
	}

}

var (
	errSink error
	paSink  *PromisedAttrs
)

func BenchmarkCommandList(b *testing.B) {
	cli := localDial(b)
	defer teardown(cli, b)
	b.ReportAllocs()
	cl := cli.BeginCommandList()
	for i := 0; i < b.N; i++ {
		paSink = cl.CurrentSong()
		paSink = cl.Status()
	}
	errSink = cl.End()
}
