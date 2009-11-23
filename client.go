// MPD (Music Player Daemon) client

package main

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

type Song struct {
	file		string;
	time		int;
	title		string;
	artist		string;
	album		string;
	track		int;
	performer	string;
	pos		int;
	id		int;
}

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

func (c *Client) getAttrs() (attrs map[string]string, err os.Error) {
	attrs = make(map[string]string);
	for {
		line, err := c.readLine();
		if err != nil {
			return nil, err
		}
		if line == "OK" {
			break
		}
		i := strings.Index(line, ": ");
		if i < 0 {
			return nil, os.NewError("can't parse line: " + line)
		}
		key := line[0:i];
		attrs[key] = line[i+2:];
	}
	return;
}

func atoi(s string) (n int) {
	n, _ = strconv.Atoi(s);
	return;
}

func (c *Client) CurrentSong() (song *Song, err os.Error) {
	c.writeLine("currentsong");
	song = new(Song);
	attrs, err := c.getAttrs();
	if err != nil {
		return nil, err
	}
	for key, val := range attrs {
		switch key {
		case "file":
			song.file = val
		case "Time":
			song.time = atoi(val)
		case "Title":
			song.title = val
		case "Artist":
			song.artist = val
		case "Album":
			song.album = val
		case "Track":
			song.track = atoi(val)
		case "Performer":
			song.performer = val
		case "Pos":
			song.pos = atoi(val)
		case "Id":
			song.id = atoi(val)
		}
	}
	return;
}

func main() {
	cli, err := Connect("127.0.0.1:6600");
	if err != nil {
		goto err
	}
	song, err := cli.CurrentSong();
	if err != nil {
		goto err
	}
	fmt.Println(song);
	cli.Disconnect();
	return;
err:
	fmt.Fprintln(os.Stderr, err);
	os.Exit(1);
}
