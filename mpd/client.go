// Copyright 2009 The GoMPD Authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

// Package mpd provides the client side interface to MPD (Music Player Daemon).
// The protocol reference can be found at http://www.musicpd.org/doc/protocol/index.html
package mpd

import (
	"errors"
	"fmt"
	"net/textproto"
	"strconv"
	"strings"
)

// Quote quotes strings in the format understood by MPD.
// See: http://git.musicpd.org/cgit/master/mpd.git/tree/src/util/Tokenizer.cxx
func quote(s string) string {
	q := make([]byte, 2+2*len(s))
	i := 0
	q[i], i = '"', i+1
	for _, c := range []byte(s) {
		if c == '"' {
			q[i], i = '\\', i+1
			q[i], i = '"', i+1
		} else {
			q[i], i = c, i+1
		}
	}
	q[i], i = '"', i+1
	return string(q[:i])
}

// Client represents a client connection to a MPD server.
type Client struct {
	text *textproto.Conn
}

// Attrs is a set of attributes returned by MPD.
type Attrs map[string]string

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

// We are reimplemeting Cmd() and PrintfLine() from textproto here, because
// the original functions append CR-LF to the end of commands. This behavior
// voilates the MPD protocol: Commands must be terminated by '\n'.
func (c *Client) cmd(format string, args ...interface{}) (uint, error) {
	id := c.text.Next()
	c.text.StartRequest(id)
	defer c.text.EndRequest(id)
	if err := c.printfLine(format, args...); err != nil {
		return 0, err
	}
	return id, nil
}

func (c *Client) printfLine(format string, args ...interface{}) error {
	fmt.Fprintf(c.text.W, format, args...)
	c.text.W.WriteByte('\n')
	return c.text.W.Flush()
}

// Close terminates the connection with MPD.
func (c *Client) Close() (err error) {
	if c.text != nil {
		c.printfLine("close")
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
	list = []string{}
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
	attrs = []Attrs{}
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
	id, err := c.cmd("currentsong")
	if err != nil {
		return nil, err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)
	return c.readAttrs("OK")
}

// Status returns information about the current status of MPD.
func (c *Client) Status() (Attrs, error) {
	id, err := c.cmd("status")
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
	id, err := c.cmd(format, args...)
	if err != nil {
		return err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)
	return c.readOKLine("OK")
}

func (c *Client) idle(subsystems ...string) ([]string, error) {
	id, err := c.cmd("idle %s", strings.Join(subsystems, " "))
	if err != nil {
		return nil, err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)
	return c.readList("changed")
}

func (c *Client) noIdle() (err error) {
	id, err := c.cmd("noidle")
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
		id, err := c.cmd("playlistinfo %d", start)
		if err != nil {
			return nil, err
		}
		c.text.StartResponse(id)
		defer c.text.EndResponse(id)
		return c.readAttrsList("file")
	}
	id, err := c.cmd("playlistinfo")
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
	return c.okCmd("delete %d:%d", start, end)
}

// DeleteId deletes the song identified by id.
func (c *Client) DeleteId(id int) error {
	return c.okCmd("deleteid %d", id)
}

// MoveId moves songid to position on the plyalist.
func (c *Client) MoveId(songid, position int) error {
	return c.okCmd("moveid %d %d", songid, position)
}

// Move moves songs in the playlist. If both start and end are positive,
// it moves those at positions in range [start, end). If end is negative,
// it moves only the song at position start.
func (c *Client) Move(start, end, to int) error {
	if end == -1 {
		return c.okCmd("move %d %d", start, to)
	}
	return c.okCmd("move %d:%d %d", start, end, to)
}

// Add adds the file/directory uri to playlist. Directories add recursively.
func (c *Client) Add(uri string) error {
	return c.okCmd("add %s", quote(uri))
}

// AddId adds the file/directory uri to playlist and returns the identity
// id of the song added. If pos is positive, the song is added to position
// pos.
func (c *Client) AddId(uri string, pos int) (int, error) {
	var id uint
	var err error
	if pos >= 0 {
		id, err = c.cmd("addid %s %d", quote(uri), pos)
	}
	id, err = c.cmd("addid %s", quote(uri))
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

// GetFiles returns the entire list of files in MPD database.
func (c *Client) GetFiles() ([]string, error) {
	id, err := c.cmd("list file")
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
func (c *Client) Update(uri string) (jobID int, err error) {
	id, err := c.cmd("update %s", quote(uri))
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
	jobID, err = strconv.Atoi(line[13:])
	if err != nil {
		return
	}
	return jobID, c.readOKLine("OK")
}

// ListAllInfo returns attributes for songs in the library. Information about
// any song that is either inside or matches the passed in uri is returned.
// To get information about every song in the library, pass in "/". 
func (c *Client) ListAllInfo(uri string) ([]Attrs, error) {
	id, err := c.cmd("listallinfo \"%s\" ", uri)
	if err != nil {
		return nil, err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)

	attrs := []Attrs{}
	for {
		line, err := c.text.ReadLine()
		if err != nil {
			return nil, err
		}
		if line == "OK" {
			break
		}
		if strings.HasPrefix(line, "file: ") { // new entry begins
			attrs = append(attrs, Attrs{})
		}
		if len(attrs) == 0 {
			if strings.HasPrefix(line, "directory: ") {
				continue
			}
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

// Stored playlists related commands

// ListPlaylists lists all stored playlists.
func (c *Client) ListPlaylists() ([]Attrs, error) {
	id, err := c.cmd("listplaylists")
	if err != nil {
		return nil, err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)
	return c.readAttrsList("playlist")
}

// PlaylistContents returns a list of attributes for songs in the specified
// stored playlist.
func (c *Client) PlaylistContents(name string) ([]Attrs, error) {
	id, err := c.cmd("listplaylistinfo %s", quote(name))
	if err != nil {
		return nil, err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)
	return c.readAttrsList("file")
}

// PlaylistLoad loads the specfied playlist into the current queue.
// If start and end are non-negative, only songs in this range are loaded.
func (c *Client) PlaylistLoad(name string, start, end int) error {
	if start < 0 || end < 0 {
		return c.okCmd("load %s", quote(name))
	}
	return c.okCmd("load %s %d:%d", quote(name), start, end)
}

// PlaylistAdd adds a song identified by uri to a stored playlist identified
// by name.
func (c *Client) PlaylistAdd(name string, uri string) error {
	return c.okCmd("playlistadd %s %s", quote(name), quote(uri))
}

// PlaylistClear clears the specified playlist.
func (c *Client) PlaylistClear(name string) error {
	return c.okCmd("playlistclear %s", quote(name))
}

// PlaylistDelete deletes the song at position pos from the specified playlist.
func (c *Client) PlaylistDelete(name string, pos int) error {
	return c.okCmd("playlistdelete %s %d", quote(name), pos)
}

// PlaylistMove moves a song identified by id in a playlist identified by name
// to the position pos.
func (c *Client) PlaylistMove(name string, id, pos int) error {
	return c.okCmd("playlistmove %s %d %d", quote(name), id, pos)
}

// PlaylistRename renames the playlist identified by name to newName.
func (c *Client) PlaylistRename(name, newName string) error {
	return c.okCmd("rename %s %s", quote(name), quote(newName))
}

// PlaylistRemove removes the playlist identified by name from the playlist
// directory.
func (c *Client) PlaylistRemove(name string) error {
	return c.okCmd("rm %s", quote(name))
}

// PlaylistSave saves the current playlist as name in the playlist directory.
func (c *Client) PlaylistSave(name string) error {
	return c.okCmd("save %s", quote(name))
}
