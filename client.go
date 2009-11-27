// Copyright © 2009 Fazlul Shahriar <fshahriar@gmail.com>.
// See LICENSE file for license details.

// This package provides the client side interface to MPD (Music Player Daemon).
// The protocol reference can be found at http://www.musicpd.org/doc/protocol/index.html
package mpd

import (
	"bufio";
	"fmt";
	"net";
	"os";
	"strconv";
	"strings";
)

var Chatty bool	// print all conversation with MPD (for debugging)

type Client struct {
	conn	net.Conn;
	rw	*bufio.ReadWriter;
}

type Attrs map[string]string

// Connect connects to MPD listening on address addr (e.g. "127.0.0.1:6600")
// on network network (e.g. "tcp").
func Connect(network, addr string) (c *Client, err os.Error) {
	conn, err := net.Dial(network, "", addr);
	if err != nil {
		return nil, err
	}
	c = new(Client);
	c.rw = bufio.NewReadWriter(bufio.NewReader(conn),
		bufio.NewWriter(conn));
	line, err := c.readLine();
	if err != nil {
		return nil, err
	}
	if line[0:6] != "OK MPD" {
		return nil, os.NewError("no greeting")
	}
	return;
}

// Close terminates the connection with MPD.
func (c *Client) Close() (err os.Error) {
	if c.conn != nil {
		c.writeLine("close");
		err = c.conn.Close();
		c.conn = nil;
	}
	return;
}

func (c *Client) readLine() (line string, err os.Error) {
	line, err = c.rw.ReadString('\n');
	if err != nil {
		return
	}
	if line[len(line)-1] == '\n' {
		line = line[0 : len(line)-1]
	}
	if Chatty {
		fmt.Println("-->", line)
	}
	return;
}

func (c *Client) writeLine(line string) (err os.Error) {
	if Chatty {
		fmt.Println("<--", line)
	}
	_, err = c.rw.Write(strings.Bytes(line + "\n"));
	// TODO: try again if # written != len(buf)
	c.rw.Flush();
	return;
}

func (c *Client) readPlaylist() (pls []Attrs, err os.Error) {
	pls = make([]Attrs, 100);

	n := 0;
	for {
		line, err := c.readLine();
		if err != nil {
			return nil, err
		}
		if line == "OK" {
			break
		}
		if strings.HasPrefix(line, "file:") {	// new song entry begins
			n++;
			if n > len(pls) || n > cap(pls) {
				pls1 := make([]Attrs, 2*cap(pls));
				for k, a := range pls {
					pls1[k] = a
				}
				pls = pls1;
			}
			pls[n-1] = make(Attrs);
		}
		if n == 0 {
			return nil, os.NewError("unexpected: " + line)
		}
		z := strings.Index(line, ": ");
		if z < 0 {
			return nil, os.NewError("can't parse line: " + line)
		}
		key := line[0:z];
		pls[n-1][key] = line[z+2:];
	}
	return pls[0:n], nil;
}

func (c *Client) getAttrs() (attrs Attrs, err os.Error) {
	attrs = make(Attrs);
	for {
		line, err := c.readLine();
		if err != nil {
			return nil, err
		}
		if line == "OK" {
			break
		}
		z := strings.Index(line, ": ");
		if z < 0 {
			return nil, os.NewError("can't parse line: " + line)
		}
		key := line[0:z];
		attrs[key] = line[z+2:];
	}
	return;
}

// CurrentSong returns information about the current song in the playlist.
func (c *Client) CurrentSong() (Attrs, os.Error) {
	c.writeLine("currentsong");
	return c.getAttrs();
}

// Status returns information about the current status of MPD.
func (c *Client) Status() (Attrs, os.Error) {
	c.writeLine("status");
	return c.getAttrs();
}

func (c *Client) readErr() (err os.Error) {
	line, err := c.readLine();
	switch {
	case err != nil:
		return err
	case line == "OK":
		return nil
	case strings.HasPrefix(line, "ACK "):
		return os.NewError(line[4:])
	}
	return os.NewError("unexpected response: " + line);
}

//
// Playback control
//

// Next plays next song in the playlist.
func (c *Client) Next() os.Error {
	c.writeLine("next");
	return c.readErr();
}

// Pause pauses playback if pause is true; resumes playback otherwise.
func (c *Client) Pause(pause bool) os.Error {
	if pause {
		c.writeLine("pause 1")
	} else {
		c.writeLine("pause 0")
	}
	return c.readErr();
}

// Play starts playing the song at playlist position pos. If pos is negative,
// start playing at the current position in the playlist.
func (c *Client) Play(pos int) os.Error {
	if pos < 0 {
		c.writeLine("play")
	} else {
		c.writeLine(fmt.Sprintf("play %d", pos))
	}
	return c.readErr();
}

// PlayId plays the song identified by id. If id is negative, start playing
// at the currect position in playlist.
func (c *Client) PlayId(id int) os.Error {
	if id < 0 {
		c.writeLine("playid")
	} else {
		c.writeLine(fmt.Sprintf("playid %d", id))
	}
	return c.readErr();
}

// Previous plays previous song in the playlist.
func (c *Client) Previous() os.Error {
	c.writeLine("next");
	return c.readErr();
}

// Seek seeks to the position time (in seconds) of the song at playlist position pos.
func (c *Client) Seek(pos, time int) os.Error {
	c.writeLine(fmt.Sprintf("seek %d %d", pos, time));
	return c.readErr();
}

// SeekId is identical to Seek except the song is identified by it's id
// (not position in playlist).
func (c *Client) SeekId(id, time int) os.Error {
	c.writeLine(fmt.Sprintf("seekid %d %d", id, time));
	return c.readErr();
}

// Stop stops playback.
func (c *Client) Stop() os.Error {
	c.writeLine("stop");
	return c.readErr();
}

//
// Playlist related functions
//

// PlaylistInfo returns attributes for songs in the current playlist. If
// both start and end are negative, it does this for all songs in
// playlist. If end is negative but start is positive, it does it for the
// song at position start. If both start and end are positive, it does it
// for positions in range [start, end).
func (c *Client) PlaylistInfo(start, end int) (pls []Attrs, err os.Error) {
	if start < 0 && end >= 0 {
		return nil, os.NewError("negative start index")
	}
	if start >= 0 && end < 0 {
		c.writeLine(fmt.Sprintf("playlistinfo %d", start));
		return c.readPlaylist();
	}
	c.writeLine("playlistinfo");
	pls, err = c.readPlaylist();
	if err != nil || start < 0 || end < 0 {
		return
	}
	return pls[start:end], nil;
}

// Delete deletes songs from playlist. If both start and end are positive,
// it deletes those at positions in range [start, end). If end is negative,
// it deletes the song at position start.
func (c *Client) Delete(start, end int) os.Error {
	if start < 0 {
		return os.NewError("negative start index")
	}
	if end < 0 {
		c.writeLine(fmt.Sprintf("delete %d", start))
	} else {
		c.writeLine(fmt.Sprintf("delete %d %d", start, end))
	}
	return c.readErr();
}

// DeleteId deletes the song identified by id.
func (c *Client) DeleteId(id int) os.Error {
	c.writeLine(fmt.Sprintf("deleteid %d", id));
	return c.readErr();
}

// Add adds the file/directory uri to playlist. Directories add recursively.
func (c *Client) Add(uri string) os.Error {
	c.writeLine(fmt.Sprintf("add %q", uri));
	return c.readErr();
}

// AddId adds the file/directory uri to playlist and returns the identity
// id of the song added. If pos is positive, the song is added to position
// pos.
func (c *Client) AddId(uri string, pos int) (id int, err os.Error) {
	if pos >= 0 {
		c.writeLine(fmt.Sprintf("addid %q %d", uri, pos))
	} else {
		c.writeLine(fmt.Sprintf("addid %q", uri))
	}
	attrs, err := c.getAttrs();
	if err != nil {
		return
	}
	tok, ok := attrs["Id"];
	if !ok {
		return -1, os.NewError("addid did not return Id")
	}
	return strconv.Atoi(tok);
}

// Clear clears the currect playlist.
func (c *Client) Clear() os.Error {
	c.writeLine("clear");
	return c.readErr();
}
