// Copyright 2009 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

// Package mpd provides the client side interface to MPD (Music Player Daemon).
// The protocol reference can be found at http://www.musicpd.org/doc/protocol/index.html
package mpd

import (
	"errors"
	"net/textproto"
	"strconv"
	"strings"
)

// Client represents a client connection to a MPD server.
type Client struct {
	text *textproto.Conn
}

// Attrs is a set of attributes returned by MPD.
type Attrs map[string]string

// Playlist is the name of a stored playlist.
type Playlist string

// Dial connects to MPD listening on address addr (e.g. "127.0.0.1:6600")
// on network network (e.g. "tcp").
func Dial(network, addr string) (c *Client, err error) {
	text, err := textproto.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	line, err := text.ReadLine()
	if err != nil {
		return nil, err
	}
	if line[0:6] != "OK MPD" {
		return nil, textproto.ProtocolError("no greeting")
	}
	return &Client{text: text}, nil
}

// DialAuthenticated connects to MPD listening on address addr (e.g. "127.0.0.1:6600")
// on network network (e.g. "tcp"). It then authenticates with MPD
// using the plaintext password password if it's not empty.
func DialAuthenticated(network, addr, password string) (c *Client, err error) {
	c, err = Dial(network, addr)
	if err == nil && len(password) > 0 {
		err = c.okCmd("password %s", password)
	}
	return c, err
}

// Close terminates the connection with MPD.
func (c *Client) Close() (err error) {
	if c.text != nil {
		c.text.PrintfLine("close")
		err = c.text.Close()
		c.text = nil
	}
	return
}

// Ping sends a no-op message to MPD. It's useful for keeping the connection alive.
func (c *Client) Ping() error {
	return c.okCmd("ping")
}

func (c *Client) readList(key string) (list []string, err error) {
	key += ": "
	for {
		line, err := c.text.ReadLine()
		if err != nil {
			return nil, err
		}
		if line == "OK" {
			break
		}
		if !strings.HasPrefix(line, key) {
			return nil, textproto.ProtocolError("unexpected: " + line)
		}
		list = append(list, line[len(key):])
	}
	return
}

func (c *Client) readAttrsList(startKey string) (attrs []Attrs, err error) {
	startKey += ": "
	for {
		line, err := c.text.ReadLine()
		if err != nil {
			return nil, err
		}
		if line == "OK" {
			break
		}
		if strings.HasPrefix(line, startKey) { // new entry begins
			attrs = append(attrs, Attrs{})
		}
		if len(attrs) == 0 {
			return nil, textproto.ProtocolError("unexpected: " + line)
		}
		i := strings.Index(line, ": ")
		if i < 0 {
			return nil, textproto.ProtocolError("can't parse line: " + line)
		}
		attrs[len(attrs)-1][line[0:i]] = line[i+2:]
	}
	return attrs, nil
}

func (c *Client) readAttrs(terminator string) (attrs Attrs, err error) {
	attrs = make(Attrs)
	for {
		line, err := c.text.ReadLine()
		if err != nil {
			return nil, err
		}
		if line == terminator {
			break
		}
		z := strings.Index(line, ": ")
		if z < 0 {
			return nil, textproto.ProtocolError("can't parse line: " + line)
		}
		key := line[0:z]
		attrs[key] = line[z+2:]
	}
	return
}

// CurrentSong returns information about the current song in the playlist.
func (c *Client) CurrentSong() (Attrs, error) {
	id, err := c.text.Cmd("currentsong")
	if err != nil {
		return nil, err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)
	return c.readAttrs("OK")
}

// Status returns information about the current status of MPD.
func (c *Client) Status() (Attrs, error) {
	id, err := c.text.Cmd("status")
	if err != nil {
		return nil, err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)
	return c.readAttrs("OK")
}

func (c *Client) readOKLine(terminator string) (err error) {
	line, err := c.text.ReadLine()
	if err != nil {
		return
	}
	if line == terminator {
		return nil
	}
	return textproto.ProtocolError("unexpected response: " + line)
}

func (c *Client) okCmd(format string, args ...interface{}) error {
	id, err := c.text.Cmd(format, args...)
	if err != nil {
		return err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)
	return c.readOKLine("OK")
}

func (c *Client) idle(subsystems ...string) ([]string, error) {
	id, err := c.text.Cmd("idle %s", strings.Join(subsystems, " "))
	if err != nil {
		return nil, err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)
	return c.readList("changed")
}

func (c *Client) noIdle() (err error) {
	id, err := c.text.Cmd("noidle")
	if err == nil {
		c.text.StartResponse(id)
		c.text.EndResponse(id)
	}
	return
}

//
// Playback control
//

// Next plays next song in the playlist.
func (c *Client) Next() error {
	return c.okCmd("next")
}

// Pause pauses playback if pause is true; resumes playback otherwise.
func (c *Client) Pause(pause bool) error {
	if pause {
		return c.okCmd("pause 1")
	}
	return c.okCmd("pause 0")
}

// Play starts playing the song at playlist position pos. If pos is negative,
// start playing at the current position in the playlist.
func (c *Client) Play(pos int) error {
	if pos < 0 {
		c.okCmd("play")
	}
	return c.okCmd("play %d", pos)
}

// PlayId plays the song identified by id. If id is negative, start playing
// at the current position in playlist.
func (c *Client) PlayId(id int) error {
	if id < 0 {
		return c.okCmd("playid")
	}
	return c.okCmd("playid %d", id)
}

// Previous plays previous song in the playlist.
func (c *Client) Previous() error {
	return c.okCmd("previous")
}

// Seek seeks to the position time (in seconds) of the song at playlist position pos.
func (c *Client) Seek(pos, time int) error {
	return c.okCmd("seek %d %d", pos, time)
}

// SeekId is identical to Seek except the song is identified by it's id
// (not position in playlist).
func (c *Client) SeekId(id, time int) error {
	return c.okCmd("seekid %d %d", id, time)
}

// Stop stops playback.
func (c *Client) Stop() error {
	return c.okCmd("stop")
}

// SetVolume sets the volume to volume. The range of volume is 0-100.
func (c *Client) SetVolume(volume int) error {
	return c.okCmd("setvol %d", volume)
}

// Random enables random playback, if random is true, disables it otherwise.
func (c *Client) Random(random bool) error {
	if random {
		return c.okCmd("random 1")
	}
	return c.okCmd("random 0")
}

// Repeat enables repeat mode, if repeat is true, disables it otherwise.
func (c *Client) Repeat(repeat bool) error {
	if repeat {
		return c.okCmd("repeat 1")
	}
	return c.okCmd("repeat 0")
}

//
// Playlist related functions
//

// PlaylistInfo returns attributes for songs in the current playlist. If
// both start and end are negative, it does this for all songs in
// playlist. If end is negative but start is positive, it does it for the
// song at position start. If both start and end are positive, it does it
// for positions in range [start, end).
func (c *Client) PlaylistInfo(start, end int) (pls []Attrs, err error) {
	if start < 0 && end >= 0 {
		return nil, errors.New("negative start index")
	}
	if start >= 0 && end < 0 {
		id, err := c.text.Cmd("playlistinfo %d", start)
		if err != nil {
			return nil, err
		}
		c.text.StartResponse(id)
		defer c.text.EndResponse(id)
		return c.readAttrsList("file")
	}
	id, err := c.text.Cmd("playlistinfo")
	if err != nil {
		return nil, err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)
	pls, err = c.readAttrsList("file")
	if err != nil || start < 0 || end < 0 {
		return
	}
	return pls[start:end], nil
}

// Delete deletes songs from playlist. If both start and end are positive,
// it deletes those at positions in range [start, end). If end is negative,
// it deletes the song at position start.
func (c *Client) Delete(start, end int) error {
	if start < 0 {
		return errors.New("negative start index")
	}
	if end < 0 {
		return c.okCmd("delete %d", start)
	}
	return c.okCmd("delete %d %d", start, end)
}

// DeleteId deletes the song identified by id.
func (c *Client) DeleteId(id int) error {
	return c.okCmd("deleteid %d", id)
}

// MoveId moves songid to position on the plyalist.
func (c *Client) MoveId(songid, position int) error {
	return c.okCmd("moveid %d %d", songid, position)
}

// Add adds the file/directory uri to playlist. Directories add recursively.
func (c *Client) Add(uri string) error {
	return c.okCmd("add %q", uri)
}

// AddId adds the file/directory uri to playlist and returns the identity
// id of the song added. If pos is positive, the song is added to position
// pos.
func (c *Client) AddId(uri string, pos int) (int, error) {
	var id uint
	var err error
	if pos >= 0 {
		id, err = c.text.Cmd("addid %q %d", uri, pos)
	}
	id, err = c.text.Cmd("addid %q", uri)
	if err != nil {
		return -1, err
	}

	c.text.StartResponse(id)
	defer c.text.EndResponse(id)

	attrs, err := c.readAttrs("OK")
	if err != nil {
		return -1, err
	}
	tok, ok := attrs["Id"]
	if !ok {
		return -1, textproto.ProtocolError("addid did not return Id")
	}
	return strconv.Atoi(tok)
}

// Clear clears the current playlist.
func (c *Client) Clear() error {
	return c.okCmd("clear")
}

// Shuffle shuffles the tracks from postion start to position end in the
// current playlist. If start or end is negative, the whole playlist is
// shuffled.
func (c *Client) Shuffle(start, end int) error {
	if start < 0 || end < 0 {
		return c.okCmd("shuffle")
	}
	return c.okCmd("shuffle %d:%d", start, end)
}

// Database related commands

// Retrieve the entire list of files
func (c *Client) GetFiles() ([]string, error) {
	id, err := c.text.Cmd("list file")
	if err != nil {
		return nil, err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)
	return c.readList("file")
}

// Update updates MPD's database: find new files, remove deleted files, update
// modified files. uri is a particular directory or file to update. If it is an
// empty string, everything is updated.
//
// The returned jobId identifies the update job, enqueued by MPD.
func (c *Client) Update(uri string) (jobId int, err error) {
	id, err := c.text.Cmd("update %q", uri)
	if err != nil {
		return
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)

	line, err := c.text.ReadLine()
	if err != nil {
		return
	}
	if !strings.HasPrefix(line, "updating_db: ") {
		return 0, textproto.ProtocolError("unexpected response: " + line)
	}
	jobId, err = strconv.Atoi(line[13:])
	if err != nil {
		return
	}
	return jobId, c.readOKLine("OK")
}

// Stored playlists related commands

// ListPlaylists lists all stored playlists.
func (c *Client) ListPlaylists() ([]Attrs, error) {
	id, err := c.text.Cmd("listplaylists")
	if err != nil {
		return nil, err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)
	return c.readAttrsList("playlist")
}

// PlaylistContents returns a list of attributes for songs in the specified
// stored playlist.
func (c *Client) PlaylistContents(name Playlist) ([]Attrs, error) {
	id, err := c.text.Cmd("listplaylistinfo %q", name)
	if err != nil {
		return nil, err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)
	return c.readAttrsList("file")
}

// PlaylistLoad loads the specfied playlist into the current queue.
// If start and end are non-negative, only songs in this range are loaded.
func (c *Client) PlaylistLoad(name Playlist, start, end int) error {
	if start < 0 || end < 0 {
		return c.okCmd("load %q", name)
	}
	return c.okCmd("load %q %d:%d", name, start, end)
}

// PlaylistAdd adds a song identified by uri to a stored playlist identified
// by name.
func (c *Client) PlaylistAdd(name Playlist, uri string) error {
	return c.okCmd("playlistadd %q %q", name, uri)
}

// PlaylistClear clears the specified playlist.
func (c *Client) PlaylistClear(name Playlist) error {
	return c.okCmd("playlistclear %q", name)
}

// PlaylistDelete deletes the song at position pos from the specified playlist.
func (c *Client) PlaylistDelete(name Playlist, pos int) error {
	return c.okCmd("playlistdelete %q %d", name, pos)
}

// PlaylistMove moves a song identified by id in a playlist identified by name
// to the position pos.
func (c *Client) PlaylistMove(name Playlist, id, pos int) error {
	return c.okCmd("playlistmove %q %d %d", name, id, pos)
}

// PlaylistRename renames the playlist identified by name to newName.
func (c *Client) PlaylistRename(name, newName Playlist) error {
	return c.okCmd("rename %q %q", name, newName)
}

// PlaylistRemove removes the playlist identified by name from the playlist
// directory.
func (c *Client) PlaylistRemove(name Playlist) error {
	return c.okCmd("rm %q", name)
}

// PlaylistSave saves the current playlist as name in the playlist directory.
func (c *Client) PlaylistSave(name Playlist) error {
	return c.okCmd("save %q", name)
}
