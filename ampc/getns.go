// Transliterated from /usr/local/plan9/src/lib9/getns.c

/*
Copyright 2001-2007 Russ Cox.  All Rights Reserved.
Copyright 2010 Fazlul Shahriar.  All Rights Reserved.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package main

import "fmt"
import "os"
import "strings"
import "unicode"

// TODO: os.Getuid followed by lookup in /etc/passwd
func getuser() (user string) {
	user, err := os.Getenverror("LOGNAME")
	if err != nil {
		return "none"
	}
	return user
}

func isme(uid uint32) bool { return uint32(os.Getuid()) == uid }

/*
 * Absent other hints, it works reasonably well to use
 * the X11 display name as the name space identifier.
 * This is how sam's B has worked since the early days.
 * Since most programs using name spaces are also using X,
 * this still seems reasonable.  Terminal-only sessions
 * can set $NAMESPACE.
 */
func nsfromdisplay() (ns string, err os.Error) {
	disp, err := os.Getenverror("DISPLAY")
	if err != nil {
		return "", os.NewError("$DISPLAY not set")
	}

	/* canonicalize: xxx:0.0 => xxx:0 */
	i := strings.Index(disp, ":")
	if i >= 0 {
		i++
		for i < len(disp) && unicode.IsDigit(int(disp[i])) {
			i++
		}
		if disp[i:] == ".0" {
			disp = disp[0:i]
		}
	}

	p := fmt.Sprintf("/tmp/ns.%s.%s", getuser(), disp)
	d, err := os.Stat(p)
	if e, ok := err.(*os.PathError); ok && e.Error == os.ENOENT {
		err = os.Mkdir(p, 0700)
		if err != nil {
			return "", err
		}
		d, err = os.Stat(p)
	}
	if err != nil {
		return "", err
	}

	if !d.IsDirectory() {
		return "", os.NewError(p + " is not a directory")
	}
	if d.Permission()&0777 != 0700 || !isme(d.Uid) {
		return "", os.NewError("bad name space dir " + p)
	}
	return p, nil
}

// Getns returns the path ns to plan9port's name space directory.
func Getns() (ns string, err os.Error) {
	ns, err = os.Getenverror("NAMESPACE")
	if err != nil {
		ns, err = nsfromdisplay()
	}
	if err != nil {
		return "", os.NewError("$NAMESPACE not set, " + err.String())
	}
	return ns, nil
}
