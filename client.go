// MPD (Music Player Daemon) client library
// Protocol Reference: http://www.musicpd.org/doc/protocol/index.html

package main

import (
	"bufio";
	"fmt";
	"net";
	"os";
	"strings";
)

type Client struct {
	conn	net.Conn;
	rw	*bufio.ReadWriter;
}

type Attrs map[string]string
type SongID int		// song identifier
type SongPOS int	// song position in the current playlist
type SongIDPOS int	// SongID or SongPOS

func Connect(addr string) (c *Client, err os.Error) {
	conn, err := net.Dial("tcp", "", addr);
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

func (c *Client) Disconnect() {
	if c.conn != nil {
		c.conn.Close()
	}
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
func (c *Client) Next() (err os.Error) {
	c.writeLine("next");
	return c.readErr();
}

// Pause pauses playback if pause is true; resumes playback otherwise.
func (c *Client) Pause(pause bool) (err os.Error) {
	if pause {
		c.writeLine("pause 1")
	} else {
		c.writeLine("pause 0")
	}
	return c.readErr();
}

// Play starts playing the song identified by id. If id is negative,
// start playing at the current position in the playlist.
func (c *Client) Play(id SongIDPOS) (err os.Error) {
	if id < 0 {
		c.writeLine("play")
	} else {
		c.writeLine(fmt.Sprintf("play %d", id))
	}
	return c.readErr();
}

// Previous plays previous song in the playlist.
func (c *Client) Previous() (err os.Error) {
	c.writeLine("next");
	return c.readErr();
}

// Seek seeks to the position time (in seconds) of the song identified by id.
func (c *Client) Seek(id SongIDPOS, time int) (err os.Error) {
	c.writeLine(fmt.Sprintf("seek %d %d", id, time));
	return c.readErr();
}

// Stop stops playback.
func (c *Client) Stop() (err os.Error) {
	c.writeLine("stop");
	return c.readErr();
}

//
// Playlist related functions
//

func (c *Client) PlaylistInfo(start, end SongPOS) (pls []Attrs, err os.Error) {
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

func main() {
	cli, err := Connect("127.0.0.1:6600");
	if err != nil {
		goto err
	}
	//cli.Play(-1);
	//cli.Pause(true);
	//cli.Stop();
	pls, err := cli.PlaylistInfo(5, -1);
	if err != nil {
		goto err
	}
	for _, s := range pls {
		fmt.Printf("song: %v\n\n", s)
	}
	goto done;

	song, err := cli.CurrentSong();
	if err != nil {
		goto err
	}
	fmt.Println("current song:", song);
	status, err := cli.Status();
	if err != nil {
		goto err
	}
	fmt.Println("status:", status);
done:
	cli.Disconnect();
	return;
err:
	fmt.Fprintln(os.Stderr, err);
	os.Exit(1);
}
