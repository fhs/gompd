// MPD (Music Player Daemon) client
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

type Attrs	map[string]string

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
		i := strings.Index(line, ": ");
		if i < 0 {
			return nil, os.NewError("can't parse line: " + line)
		}
		key := line[0:i];
		attrs[key] = line[i+2:];
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

func main() {
	cli, err := Connect("127.0.0.1:6600");
	if err != nil {
		goto err
	}
	song, err := cli.CurrentSong();
	if err != nil {
		goto err
	}
	fmt.Println("current song:", song);
	status, err := cli.Status();
	if err != nil {
		goto err;
	}
	fmt.Println("status:", status);
	cli.Disconnect();
	return;
err:
	fmt.Fprintln(os.Stderr, err);
	os.Exit(1);
}
