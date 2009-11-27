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

type Client struct {
	conn	net.Conn;
	rw	*bufio.ReadWriter;
}

type Attrs map[string]string

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
	fmt.Println("-->", line);
	return;
}

func (c *Client) writeLine(line string) (err os.Error) {
	fmt.Println("<--", line);
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

func (c *Client) CurrentSong() (Attrs, os.Error) {
	c.writeLine("currentsong");
	return c.getAttrs();
}

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

func (c *Client) PlayId(id int) os.Error	{ return c.Play(id) }

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

func (c *Client) SeekId(id, time int) os.Error {
	return c.Seek(id, time)
}

// Stop stops playback.
func (c *Client) Stop() os.Error {
	c.writeLine("stop");
	return c.readErr();
}

//
// Playlist related functions
//

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

func (c *Client) DeleteId(songid int) os.Error {
	c.writeLine(fmt.Sprintf("delete %d", songid));
	return c.readErr();
}

func (c *Client) Add(uri string) os.Error {
	c.writeLine(fmt.Sprintf("%q", uri));
	return c.readErr();
}

func (c *Client) AddId(uri string, pos int) (id int, err os.Error) {
	if pos >= 0 {
		c.writeLine(fmt.Sprintf("%q %d", uri, pos))
	} else {
		c.writeLine(fmt.Sprintf("%q", uri))
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

func (c *Client) Clear() os.Error {
	c.writeLine("clear");
	return c.readErr();
}
