// Copyright 2013 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package mpd

import (
	"container/list"
	"errors"
	"fmt"
	"strconv"
)

type cmdType uint

const (
	cmd_no_return cmdType = iota
	cmd_attr_return
	cmd_id_return
)

type command struct {
	cmd     string
	promise interface{}
	typeOf  cmdType
}

// CommandList is for batch/mass MPD commands.
// See http://www.musicpd.org/doc/protocol/ch01s04.html
// for more details.
type CommandList struct {
	client *Client
	cmdQ   *list.List
}

// PromisedAttrs is a set of promised attributes (to be) returned by MPD.
type PromisedAttrs struct {
	attrs    Attrs
	computed bool
}

func newPromisedAttrs() *PromisedAttrs {
	return &PromisedAttrs{attrs: make(Attrs), computed: false}
}

// PromisedId is a promised identifier (to be) returned by MPD.
type PromisedId int

// Value is a convenience method for ensuring that a promise
// has been computed, returning the Attrs.
func (pa *PromisedAttrs) Value() (Attrs, error) {
	if !pa.computed {
		return nil, errors.New("This value has not been computed yet.")
	}
	return pa.attrs, nil
}

// Value is a convenience method for ensuring that a promise
// has been computed, returning the ID.
func (pi *PromisedId) Value() (int, error) {
	if *pi == -1 {
		return -1, errors.New("This value has not been computed yet.")
	}
	return (int)(*pi), nil
}

// BeginCommandList creates a new CommandList structure using
// this connection.
func (c *Client) BeginCommandList() *CommandList {
	return &CommandList{c, list.New()}
}

// Ping sends a no-op message to MPD. It's useful for keeping the connection alive.
func (cl *CommandList) Ping() {
	cl.cmdQ.PushBack(&command{"ping", nil, cmd_no_return})
}

// CurrentSong returns information about the current song in the playlist.
func (cl *CommandList) CurrentSong() *PromisedAttrs {
	pa := newPromisedAttrs()
	cl.cmdQ.PushBack(&command{"currentsong", pa, cmd_attr_return})
	return pa
}

// Status returns information about the current status of MPD.
func (cl *CommandList) Status() *PromisedAttrs {
	pa := newPromisedAttrs()
	cl.cmdQ.PushBack(&command{"status", pa, cmd_attr_return})
	return pa
}

//
// Playback control
//

// Next plays next song in the playlist.
func (cl *CommandList) Next() {
	cl.cmdQ.PushBack(&command{"next", nil, cmd_no_return})
}

// Pause pauses playback if pause is true; resumes playback otherwise.
func (cl *CommandList) Pause(pause bool) {
	if pause {
		cl.cmdQ.PushBack(&command{"pause 1", nil, cmd_no_return})
	} else {
		cl.cmdQ.PushBack(&command{"pause 0", nil, cmd_no_return})
	}
}

// Play starts playing the song at playlist position pos. If pos is negative,
// start playing at the current position in the playlist.
func (cl *CommandList) Play(pos int) {
	if pos < 0 {
		cl.cmdQ.PushBack(&command{"play", nil, cmd_no_return})
	} else {
		cl.cmdQ.PushBack(&command{fmt.Sprintf("play %d", pos), nil, cmd_no_return})
	}
}

// PlayId plays the song identified by id. If id is negative, start playing
// at the currect position in playlist.
func (cl *CommandList) PlayId(id int) {
	if id < 0 {
		cl.cmdQ.PushBack(&command{"playid", nil, cmd_no_return})
	} else {
		cl.cmdQ.PushBack(&command{fmt.Sprintf("playid %d", id), nil, cmd_no_return})
	}
}

// Previous plays previous song in the playlist.
func (cl *CommandList) Previous() {
	cl.cmdQ.PushBack(&command{"previous", nil, cmd_no_return})
}

// Seek seeks to the position time (in seconds) of the song at playlist position pos.
func (cl *CommandList) Seek(pos, time int) {
	cl.cmdQ.PushBack(&command{fmt.Sprintf("seek %d %d", pos, time), nil, cmd_no_return})
}

// SeekId is identical to Seek except the song is identified by it's id
// (not position in playlist).
func (cl *CommandList) SeekId(id, time int) {
	cl.cmdQ.PushBack(&command{fmt.Sprintf("seek %d %d", id, time), nil, cmd_no_return})
}

// Stop stops playback.
func (cl *CommandList) Stop() {
	cl.cmdQ.PushBack(&command{"stop", nil, cmd_no_return})
}

// SetVolume sets the MPD volume level.
func (cl *CommandList) SetVolume(volume int) {
	cl.cmdQ.PushBack(&command{fmt.Sprintf("setvol %d", volume), nil, cmd_no_return})
}

// Random enables random playback, if random is true, disables it otherwise.
func (cl *CommandList) Random(random bool) {
	if random {
		cl.cmdQ.PushBack(&command{"random 1", nil, cmd_no_return})
	} else {
		cl.cmdQ.PushBack(&command{"random 0", nil, cmd_no_return})
	}
}

// Repeat enables reapeat mode, if repeat is true, disables it otherwise.
func (cl *CommandList) Repeat(repeat bool) {
	if repeat {
		cl.cmdQ.PushBack(&command{"repeat 1", nil, cmd_no_return})
	} else {
		cl.cmdQ.PushBack(&command{"repeat 0", nil, cmd_no_return})
	}
}

//
// Playlist related functions
//

// Delete deletes songs from playlist. If both start and end are positive,
// it deletes those at positions in range [start, end). If end is negative,
// it deletes the song at position start.
func (cl *CommandList) Delete(start, end int) error {
	if start < 0 {
		return errors.New("negative start index")
	}
	if end < 0 {
		cl.cmdQ.PushBack(&command{fmt.Sprintf("delete %d", start), nil, cmd_no_return})
	} else {
		cl.cmdQ.PushBack(&command{fmt.Sprintf("delete %d %d", start, end), nil, cmd_no_return})
	}
	return nil
}

// DeleteId deletes the song identified by id.
func (cl *CommandList) DeleteId(id int) {
	cl.cmdQ.PushBack(&command{fmt.Sprintf("deleteid %d", id), nil, cmd_no_return})
}

// MoveId moves songid to position on the playlist.
func (cl *CommandList) MoveId(songid, position int) {
	cl.cmdQ.PushBack(&command{fmt.Sprintf("moveid %d %d", songid, position), nil, cmd_no_return})
}

// Add adds the file/directory uri to playlist. Directories add recursively.
func (cl *CommandList) Add(uri string) {
	cl.cmdQ.PushBack(&command{fmt.Sprintf("add %s", quote(uri)), nil, cmd_no_return})
}

// AddId adds the file/directory uri to playlist and returns the identity
// id of the song added. If pos is positive, the song is added to position
// pos.
func (cl *CommandList) AddId(uri string, pos int) *PromisedId {
	var id PromisedId = -1
	if pos >= 0 {
		cl.cmdQ.PushBack(&command{fmt.Sprintf("addid %s %d", quote(uri), pos), &id, cmd_id_return})
	} else {
		cl.cmdQ.PushBack(&command{fmt.Sprintf("addid %s", quote(uri)), &id, cmd_id_return})
	}
	return &id
}

// Clear clears the current playlist.
func (cl *CommandList) Clear() {
	cl.cmdQ.PushBack(&command{"clear", nil, cmd_no_return})
}

// Shuffle shuffles the tracks from postion start to position end in the
// current playlist. If start or end is negative, the whole playlist is
// shuffled.
func (cl *CommandList) Shuffle(start, end int) {
	if start < 0 || end < 0 {
		cl.cmdQ.PushBack(&command{"shuffle", nil, cmd_no_return})
	}
	cl.cmdQ.PushBack(&command{fmt.Sprintf("shuffe %d:%d", start, end), nil, cmd_no_return})
}

// Update updates MPD's database: find new files, remove deleted files, update
// modified files. uri is a particular directory or file to update. If it is an
// empty string, everything is updated.
func (cl *CommandList) Update(uri string) (attrs *PromisedAttrs) {
	attrs = newPromisedAttrs()
	cl.cmdQ.PushBack(&command{fmt.Sprintf("update %s", quote(uri)), attrs, cmd_attr_return})
	return
}

// Stored playlists related commands.

// PlaylistLoad loads the specfied playlist into the current queue.
// If start and end are non-negative, only songs in this range are loaded.
func (cl *CommandList) PlaylistLoad(name string, start, end int) {
	if start < 0 || end < 0 {
		cl.cmdQ.PushBack(&command{fmt.Sprintf("load %s", quote(name)), nil, cmd_no_return})
	} else {
		cl.cmdQ.PushBack(&command{fmt.Sprintf("load %s %d:%d", quote(name), start, end), nil, cmd_no_return})
	}
}

// PlaylistAdd adds a song identified by uri to a stored playlist identified
// by name.
func (cl *CommandList) PlaylistAdd(name string, uri string) {
	cl.cmdQ.PushBack(&command{fmt.Sprintf("playlistadd %s %s", quote(name), quote(uri)), nil, cmd_no_return})
}

// PlaylistClear clears the specified playlist.
func (cl *CommandList) PlaylistClear(name string) {
	cl.cmdQ.PushBack(&command{fmt.Sprintf("playlistclear %s", quote(name)), nil, cmd_no_return})
}

// PlaylistDelete deletes the song at position pos from the specified playlist.
func (cl *CommandList) PlaylistDelete(name string, pos int) {
	cl.cmdQ.PushBack(&command{fmt.Sprintf("playlistdelete %s %d", quote(name), pos), nil, cmd_no_return})
}

// PlaylistMove moves a song identified by id in a playlist identified by name
// to the position pos.
func (cl *CommandList) PlaylistMove(name string, id, pos int) {
	cl.cmdQ.PushBack(&command{fmt.Sprintf("playlistmove %s %d %d", quote(name), id, pos), nil, cmd_no_return})
}

// PlaylistRename renames the playlist identified by name to newName.
func (cl *CommandList) PlaylistRename(name, newName string) {
	cl.cmdQ.PushBack(&command{fmt.Sprintf("rename %s %s", quote(name), quote(newName)), nil, cmd_no_return})
}

// PlaylistRemove removes the playlist identified by name from the playlist
// directory.
func (cl *CommandList) PlaylistRemove(name string) {
	cl.cmdQ.PushBack(&command{fmt.Sprintf("rm %s", quote(name)), nil, cmd_no_return})
}

// PlaylistSave saves the current playlist as name in the playlist directory.
func (cl *CommandList) PlaylistSave(name string) {
	cl.cmdQ.PushBack(&command{fmt.Sprintf("save %s", quote(name)), nil, cmd_no_return})
}

// End executes the command list.
func (cl *CommandList) End() error {

	// Tell MPD to start an OK command list:
	beginId, beginErr := cl.client.text.Cmd("command_list_ok_begin")
	if beginErr != nil {
		return beginErr
	}
	cl.client.text.StartResponse(beginId)
	cl.client.text.EndResponse(beginId)

	// Ensure the queue is cleared regardless.
	defer cl.cmdQ.Init()

	// Issue all of the queued up commands in the list:
	for e := cl.cmdQ.Front(); e != nil; e = e.Next() {
		cmdId, cmdErr := cl.client.text.Cmd(e.Value.(*command).cmd)
		if cmdErr != nil {
			return cmdErr
		}
		cl.client.text.StartResponse(cmdId)
		cl.client.text.EndResponse(cmdId)
	}

	// Tell MPD to end the command list and do the operations.
	endId, endErr := cl.client.text.Cmd("command_list_end")
	if endErr != nil {
		return endErr
	}
	cl.client.text.StartResponse(endId)
	defer cl.client.text.EndResponse(endId)

	// Get the responses back and check for errors:
	for e := cl.cmdQ.Front(); e != nil; e = e.Next() {
		switch e.Value.(*command).typeOf {

		case cmd_no_return:
			if err := cl.client.readOKLine("list_OK"); err != nil {
				return err
			}

		case cmd_attr_return:
			a, aErr := cl.client.readAttrs("list_OK")
			if aErr != nil {
				return aErr
			}
			pa := e.Value.(*command).promise.(*PromisedAttrs)
			pa.attrs = a
			pa.computed = true

		case cmd_id_return:
			a, aErr := cl.client.readAttrs("list_OK")
			if aErr != nil {
				return aErr
			}
			rid, ridErr := strconv.Atoi(a["Id"])
			if ridErr != nil {
				return ridErr
			}
			*(e.Value.(*command).promise.(*PromisedId)) = (PromisedId)(rid)

		}
	}

	// Finalize the command list with the last OK:
	if cerr := cl.client.readOKLine("OK"); cerr != nil {
		return cerr
	}

	return nil

}
