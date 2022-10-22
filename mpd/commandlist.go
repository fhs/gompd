// Copyright 2013 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package mpd

import (
	"errors"
	"fmt"
	"strconv"
)

type command struct {
	promise interface{}
	cmd     string
}

// CommandList is for batch/mass MPD commands.
// See http://www.musicpd.org/doc/protocol/command_lists.html
// for more details.
type CommandList struct {
	client *Client
	cmds   []*command
}

// PromisedAttrs is a set of promised attributes (to be) returned by MPD.
type PromisedAttrs struct {
	attrs    Attrs
	computed bool
}

func newPromisedAttrs() *PromisedAttrs {
	return &PromisedAttrs{attrs: make(Attrs), computed: false}
}

// PromisedID is a promised identifier (to be) returned by MPD.
type PromisedID int

// Value returns the Attrs that were computed when CommandList.End was
// called. Returns an error if CommandList.End has not yet been called.
func (pa *PromisedAttrs) Value() (Attrs, error) {
	if !pa.computed {
		return nil, errors.New("value has not been computed yet")
	}
	return pa.attrs, nil
}

// Value returns the ID that was computed when CommandList.End was
// called. Returns an error if CommandList.End has not yet been called.
func (pi *PromisedID) Value() (int, error) {
	if *pi == -1 {
		return -1, errors.New("value has not been computed yet")
	}
	return int(*pi), nil
}

// BeginCommandList creates a new CommandList structure using
// this connection.
func (c *Client) BeginCommandList() *CommandList {
	return &CommandList{client: c}
}

// Ping sends a no-op message to MPD. It's useful for keeping the connection alive.
func (cl *CommandList) Ping() {
	cl.cmds = append(cl.cmds, &command{cmd: "ping"})
}

// CurrentSong returns information about the current song in the playlist.
func (cl *CommandList) CurrentSong() *PromisedAttrs {
	pa := newPromisedAttrs()
	cl.cmds = append(cl.cmds, &command{promise: pa, cmd: "currentsong"})
	return pa
}

// Status returns information about the current status of MPD.
func (cl *CommandList) Status() *PromisedAttrs {
	pa := newPromisedAttrs()
	cl.cmds = append(cl.cmds, &command{promise: pa, cmd: "status"})
	return pa
}

//
// Playback control
//

// Next plays next song in the playlist.
func (cl *CommandList) Next() {
	cl.cmds = append(cl.cmds, &command{cmd: "next"})
}

// Pause pauses playback if pause is true; resumes playback otherwise.
func (cl *CommandList) Pause(pause bool) {
	if pause {
		cl.cmds = append(cl.cmds, &command{cmd: "pause 1"})
	} else {
		cl.cmds = append(cl.cmds, &command{cmd: "pause 0"})
	}
}

// Play starts playing the song at playlist position pos. If pos is negative,
// start playing at the current position in the playlist.
func (cl *CommandList) Play(pos int) {
	if pos < 0 {
		cl.cmds = append(cl.cmds, &command{cmd: "play"})
	} else {
		cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("play %d", pos)})
	}
}

// PlayID plays the song identified by id. If id is negative, start playing
// at the currect position in playlist.
func (cl *CommandList) PlayID(id int) {
	if id < 0 {
		cl.cmds = append(cl.cmds, &command{cmd: "playid"})
	} else {
		cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("playid %d", id)})
	}
}

// Previous plays previous song in the playlist.
func (cl *CommandList) Previous() {
	cl.cmds = append(cl.cmds, &command{cmd: "previous"})
}

// Seek seeks to the position time (in seconds) of the song at playlist position pos.
func (cl *CommandList) Seek(pos, time int) {
	cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("seek %d %d", pos, time)})
}

// SeekID is identical to Seek except the song is identified by it's id
// (not position in playlist).
func (cl *CommandList) SeekID(id, time int) {
	cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("seekid %d %d", id, time)})
}

// Stop stops playback.
func (cl *CommandList) Stop() {
	cl.cmds = append(cl.cmds, &command{cmd: "stop"})
}

// SetVolume sets the MPD volume level.
func (cl *CommandList) SetVolume(volume int) {
	cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("setvol %d", volume)})
}

// Random enables random playback, if random is true, disables it otherwise.
func (cl *CommandList) Random(random bool) {
	if random {
		cl.cmds = append(cl.cmds, &command{cmd: "random 1"})
	} else {
		cl.cmds = append(cl.cmds, &command{cmd: "random 0"})
	}
}

// Repeat enables repeat mode, if repeat is true, disables it otherwise.
func (cl *CommandList) Repeat(repeat bool) {
	if repeat {
		cl.cmds = append(cl.cmds, &command{cmd: "repeat 1"})
	} else {
		cl.cmds = append(cl.cmds, &command{cmd: "repeat 0"})
	}
}

// Single enables single song mode, if single is true, disables it otherwise.
func (cl *CommandList) Single(single bool) {
	if single {
		cl.cmds = append(cl.cmds, &command{cmd: "single 1"})
	} else {
		cl.cmds = append(cl.cmds, &command{cmd: "single 0"})
	}
}

// Consume enables consume mode, if consume is true, disables it otherwise.
func (cl *CommandList) Consume(consume bool) {
	if consume {
		cl.cmds = append(cl.cmds, &command{cmd: "consume 1"})
	} else {
		cl.cmds = append(cl.cmds, &command{cmd: "consume 0"})
	}
}

//
// Playlist related functions
//

// SetPriority sets the priority for songs in the playlist. If both start and
// end are non-negative, it updates those at positions in range [start, end).
// If end is negative, it updates the song at position start.
func (cl *CommandList) SetPriority(priority, start, end int) error {
	if start < 0 {
		return errors.New("negative start index")
	}
	if end < 0 {
		cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("prio %d %d", priority, start)})
	} else {
		cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("prio %d %d:%d", priority, start, end)})
	}
	return nil
}

// SetPriorityID sets the priority for the song identified by id.
func (cl *CommandList) SetPriorityID(priority, id int) {
	cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("prioid %d %d", priority, id)})
}

// Delete deletes songs from playlist. If both start and end are positive,
// it deletes those at positions in range [start, end). If end is negative,
// it deletes the song at position start.
func (cl *CommandList) Delete(start, end int) error {
	if start < 0 {
		return errors.New("negative start index")
	}
	if end < 0 {
		cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("delete %d", start)})
	} else {
		cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("delete %d:%d", start, end)})
	}
	return nil
}

// DeleteID deletes the song identified by id.
func (cl *CommandList) DeleteID(id int) {
	cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("deleteid %d", id)})
}

// Move moves the songs between the positions start and end to the new position
// position. If end is negative, only the song at position start is moved.
func (cl *CommandList) Move(start, end, position int) error {
	if start < 0 {
		return errors.New("negative start index")
	}
	if end < 0 {
		cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("move %d %d", start, position)})
	} else {
		cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("move %d:%d %d", start, end, position)})
	}
	return nil
}

// MoveID moves songid to position on the playlist.
func (cl *CommandList) MoveID(songid, position int) {
	cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("moveid %d %d", songid, position)})
}

// Add adds the file/directory uri to playlist. Directories add recursively.
func (cl *CommandList) Add(uri string) {
	cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("add %s", quote(uri))})
}

// AddID adds the file/directory uri to playlist and returns the identity
// id of the song added. If pos is positive, the song is added to position
// pos.
func (cl *CommandList) AddID(uri string, pos int) *PromisedID {
	var id PromisedID = -1
	if pos >= 0 {
		cl.cmds = append(cl.cmds, &command{promise: &id, cmd: fmt.Sprintf("addid %s %d", quote(uri), pos)})
	} else {
		cl.cmds = append(cl.cmds, &command{promise: &id, cmd: fmt.Sprintf("addid %s", quote(uri))})
	}
	return &id
}

// Clear clears the current playlist.
func (cl *CommandList) Clear() {
	cl.cmds = append(cl.cmds, &command{cmd: "clear"})
}

// Shuffle shuffles the tracks from position start to position end in the
// current playlist. If start or end is negative, the whole playlist is
// shuffled.
func (cl *CommandList) Shuffle(start, end int) {
	if start < 0 || end < 0 {
		cl.cmds = append(cl.cmds, &command{cmd: "shuffle"})
		return
	}
	cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("shuffle %d:%d", start, end)})
}

// Update updates MPD's database: find new files, remove deleted files, update
// modified files. uri is a particular directory or file to update. If it is an
// empty string, everything is updated.
func (cl *CommandList) Update(uri string) (attrs *PromisedAttrs) {
	attrs = newPromisedAttrs()
	cl.cmds = append(cl.cmds, &command{promise: attrs, cmd: fmt.Sprintf("update %s", quote(uri))})
	return
}

// Stored playlists related commands.

// PlaylistLoad loads the specfied playlist into the current queue.
// If start and end are non-negative, only songs in this range are loaded.
func (cl *CommandList) PlaylistLoad(name string, start, end int) {
	if start < 0 || end < 0 {
		cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("load %s", quote(name))})
	} else {
		cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("load %s %d:%d", quote(name), start, end)})
	}
}

// PlaylistAdd adds a song identified by uri to a stored playlist identified
// by name.
func (cl *CommandList) PlaylistAdd(name string, uri string) {
	cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("playlistadd %s %s", quote(name), quote(uri))})
}

// PlaylistClear clears the specified playlist.
func (cl *CommandList) PlaylistClear(name string) {
	cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("playlistclear %s", quote(name))})
}

// PlaylistDelete deletes the song at position pos from the specified playlist.
func (cl *CommandList) PlaylistDelete(name string, pos int) {
	cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("playlistdelete %s %d", quote(name), pos)})
}

// PlaylistMove moves a song identified by id in a playlist identified by name
// to the position pos.
func (cl *CommandList) PlaylistMove(name string, id, pos int) {
	cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("playlistmove %s %d %d", quote(name), id, pos)})
}

// PlaylistRename renames the playlist identified by name to newName.
func (cl *CommandList) PlaylistRename(name, newName string) {
	cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("rename %s %s", quote(name), quote(newName))})
}

// PlaylistRemove removes the playlist identified by name from the playlist
// directory.
func (cl *CommandList) PlaylistRemove(name string) {
	cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("rm %s", quote(name))})
}

// PlaylistSave saves the current playlist as name in the playlist directory.
func (cl *CommandList) PlaylistSave(name string) {
	cl.cmds = append(cl.cmds, &command{cmd: fmt.Sprintf("save %s", quote(name))})
}

// End executes the command list.
func (cl *CommandList) End() error {
	// Tell MPD to start an OK command list:
	beginID, beginErr := cl.client.cmd("command_list_ok_begin")
	if beginErr != nil {
		return beginErr
	}
	cl.client.text.StartResponse(beginID)
	cl.client.text.EndResponse(beginID)

	// Issue all of the queued up commands in the list:
	for _, cmd := range cl.cmds {
		cmdID, cmdErr := cl.client.cmd(cmd.cmd)
		if cmdErr != nil {
			return cmdErr
		}
		cl.client.text.StartResponse(cmdID)
		cl.client.text.EndResponse(cmdID)
	}

	// Tell MPD to end the command list and do the operations.
	endID, endErr := cl.client.cmd("command_list_end")
	if endErr != nil {
		return endErr
	}
	cl.client.text.StartResponse(endID)
	defer cl.client.text.EndResponse(endID)

	// Get the responses back and check for errors:
	for _, cmd := range cl.cmds {
		switch p := cmd.promise.(type) {
		case *PromisedAttrs:
			a, aErr := cl.client.readAttrs("list_OK")
			if aErr != nil {
				return aErr
			}
			p.attrs = a
			p.computed = true
		case *PromisedID:
			a, aErr := cl.client.readAttrs("list_OK")
			if aErr != nil {
				return aErr
			}
			rid, ridErr := strconv.Atoi(a["Id"])
			if ridErr != nil {
				return ridErr
			}
			*p = PromisedID(rid)
		default:
			if err := cl.client.readOKLine("list_OK"); err != nil {
				return err
			}
		}
	}

	// Finalize the command list with the last OK:
	return cl.client.readOKLine("OK")
}
