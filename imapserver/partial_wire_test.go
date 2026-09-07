package imapserver_test

import (
	"fmt"
	"strings"
	"testing"
)

// A FETCH partial whose offset and size sum past int64 used to panic inside
// extractPartial and drop the connection. The server must answer it like any
// other over-long range: the remainder of the section from the offset.
func TestFetchPartialOverflowAnswered(t *testing.T) {
	w := dialParseServer(t, nil)

	msg := "Subject: hi\r\n\r\nhello world\r\n"
	fmt.Fprintf(w.conn, "a2 APPEND INBOX {%d}\r\n", len(msg))
	if l := w.line(); !strings.HasPrefix(l, "+") {
		t.Fatalf("APPEND continuation: got %q", l)
	}
	fmt.Fprint(w.conn, msg+"\r\n")
	if got := w.until("a2"); !strings.HasPrefix(got, "a2 OK") {
		t.Fatalf("APPEND: %q", got)
	}
	fmt.Fprint(w.conn, "a3 SELECT INBOX\r\n")
	if got := w.until("a3"); !strings.HasPrefix(got, "a3 OK") {
		t.Fatalf("SELECT: %q", got)
	}

	fmt.Fprint(w.conn, "a4 FETCH 1 BODY[]<5.9223372036854775807>\r\n")
	want := msg[5:]
	var sawData bool
	for {
		l := w.line()
		if strings.HasPrefix(l, "a4 ") {
			if !strings.HasPrefix(l, "a4 OK") {
				t.Fatalf("FETCH answered %q; want OK", l)
			}
			break
		}
		if !strings.HasPrefix(l, "* 1 FETCH (") || !strings.Contains(l, " BODY[]<5> {") {
			continue
		}
		sawData = true
		if !strings.HasSuffix(l, fmt.Sprintf("{%d}", len(want))) {
			t.Errorf("literal size in %q; want %d", l, len(want))
		}
		buf := make([]byte, len(want))
		for n := 0; n < len(buf); {
			m, err := w.rd.Read(buf[n:])
			if err != nil {
				t.Fatalf("read literal: %v", err)
			}
			n += m
		}
		if string(buf) != want {
			t.Errorf("partial = %q; want %q", buf, want)
		}
	}
	if !sawData {
		t.Errorf("no BODY[]<5> data item in FETCH response")
	}
}
