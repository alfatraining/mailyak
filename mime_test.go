package mailyak

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"regexp"
	"strings"
	"testing"
)

// TestMailYakFromHeader ensures the fromHeader method returns valid headers
func TestMailYakFromHeader(t *testing.T) {
	tests := []struct {
		// Test description.
		name string
		// Receiver fields.
		rfromAddr string
		rfromName string
		// Expected results.
		want string
	}{
		{
			"With name",
			"dom@itsallbroken.com",
			"Dom",
			"From: Dom <dom@itsallbroken.com>\r\n",
		},
		{
			"Without name",
			"dom@itsallbroken.com",
			"",
			"From: dom@itsallbroken.com\r\n",
		},
		{
			"Without either",
			"",
			"",
			"From: \r\n",
		},
	}
	for _, tt := range tests {
		m := MailYak{
			fromAddr: tt.rfromAddr,
			fromName: tt.rfromName,
		}

		if got := m.fromHeader(); got != tt.want {
			t.Errorf("%q. MailYak.fromHeader() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// TestMailYakWriteHeaders ensures the Mime-Version, Reply-To, From, To and
// Subject headers are correctly wrote
func TestMailYakWriteHeaders(t *testing.T) {
	tests := []struct {
		// Test description.
		name string
		// Receiver fields.
		rtoAddrs []string
		rsubject string
		rreplyTo string
		// Expected results.
		wantBuf string
	}{
		{
			"All fields",
			[]string{"test@itsallbroken.com"},
			"Test",
			"help@itsallbroken.com",
			"From: Dom <dom@itsallbroken.com>\r\nMime-Version: 1.0\r\nReply-To: help@itsallbroken.com\r\nSubject: Test\r\nTo: test@itsallbroken.com\r\n",
		},
		{
			"No reply-to",
			[]string{"test@itsallbroken.com"},
			"",
			"",
			"From: Dom <dom@itsallbroken.com>\r\nMime-Version: 1.0\r\nSubject: \r\nTo: test@itsallbroken.com\r\n",
		},
		{
			"Multiple To addresses",
			[]string{"test@itsallbroken.com", "repairs@itsallbroken.com"},
			"",
			"",
			"From: Dom <dom@itsallbroken.com>\r\nMime-Version: 1.0\r\nSubject: \r\nTo: test@itsallbroken.com\r\nTo: repairs@itsallbroken.com\r\n",
		},
	}
	for _, tt := range tests {
		m := MailYak{
			toAddrs:  tt.rtoAddrs,
			subject:  tt.rsubject,
			fromAddr: "dom@itsallbroken.com",
			fromName: "Dom",
			replyTo:  tt.rreplyTo,
		}

		buf := &bytes.Buffer{}
		m.writeHeaders(buf)

		if gotBuf := buf.String(); gotBuf != tt.wantBuf {
			t.Errorf("%q. MailYak.writeHeaders() = %v, want %v", tt.name, gotBuf, tt.wantBuf)
		}
	}
}

// TestMailYakWriteBody ensures the correct MIME parts are wrote for the body
func TestMailYakWriteBody(t *testing.T) {
	tests := []struct {
		// Test description.
		name string
		// Receiver fields.
		rHTML  string
		rPlain string
		// Parameters.
		boundary string
		// Expected results.
		wantW   string
		wantErr bool
	}{
		{
			"Boundary name",
			"",
			"",
			"test",
			"\r\n--test--\r\n",
			false,
		},
		{
			"HTML",
			"HTML",
			"",
			"t",
			"--t\r\nContent-Type: text/html; charset=UTF-8\r\n\r\nHTML\r\n--t--\r\n",
			false,
		},
		{
			"Plain text",
			"",
			"Plain",
			"t",
			"--t\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nPlain\r\n--t--\r\n",
			false,
		},
		{
			"Both",
			"HTML",
			"Plain",
			"t",
			"--t\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nPlain\r\n--t\r\nContent-Type: text/html; charset=UTF-8\r\n\r\nHTML\r\n--t--\r\n",
			false,
		},
	}
	for _, tt := range tests {
		m := MailYak{
			html:  []byte(tt.rHTML),
			plain: []byte(tt.rPlain),
		}

		w := &bytes.Buffer{}
		if err := m.writeBody(w, tt.boundary); (err != nil) != tt.wantErr {
			t.Errorf("%q. MailYak.writeBody() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}

		if gotW := w.String(); gotW != tt.wantW {
			t.Errorf("%q. MailYak.writeBody() = %v, want %v", tt.name, gotW, tt.wantW)
		}
	}
}

// TestMailYakBuildMime tests all the other mime-related bits combine in a sane way
func TestMailYakBuildMime(t *testing.T) {
	tests := []struct {
		// Test description.
		name string
		// Receiver fields.
		rHTML     []byte
		rPlain    []byte
		rtoAddrs  []string
		rsubject  string
		rfromAddr string
		rfromName string
		rreplyTo  string
		// Expected results.
		want    string
		wantErr bool
	}{
		{
			"Empty",
			[]byte{},
			[]byte{},
			[]string{""},
			"",
			"",
			"",
			"",
			"From: \r\nMime-Version: 1.0\r\nSubject: \r\nTo: \r\nContent-Type: multipart/mixed;\r\n\tboundary=\"mixed\"; charset=UTF-8\r\n\r\n--mixed\r\nContent-Type: multipart/alternative;\r\n\tboundary=\"alt\"\r\n\r\n\r\n--alt--\r\n\r\n--mixed--\r\n",
			false,
		},
		{
			"HTML",
			[]byte("HTML"),
			[]byte{},
			[]string{""},
			"",
			"",
			"",
			"",
			"From: \r\nMime-Version: 1.0\r\nSubject: \r\nTo: \r\nContent-Type: multipart/mixed;\r\n\tboundary=\"mixed\"; charset=UTF-8\r\n\r\n--mixed\r\nContent-Type: multipart/alternative;\r\n\tboundary=\"alt\"\r\n\r\n--alt\r\nContent-Type: text/html; charset=UTF-8\r\n\r\nHTML\r\n--alt--\r\n\r\n--mixed--\r\n",
			false,
		},
		{
			"Plain",
			[]byte{},
			[]byte("Plain"),
			[]string{""},
			"",
			"",
			"",
			"",
			"From: \r\nMime-Version: 1.0\r\nSubject: \r\nTo: \r\nContent-Type: multipart/mixed;\r\n\tboundary=\"mixed\"; charset=UTF-8\r\n\r\n--mixed\r\nContent-Type: multipart/alternative;\r\n\tboundary=\"alt\"\r\n\r\n--alt\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nPlain\r\n--alt--\r\n\r\n--mixed--\r\n",
			false,
		},
		{
			"Reply-To",
			[]byte{},
			[]byte{},
			[]string{""},
			"",
			"",
			"",
			"reply",
			"From: \r\nMime-Version: 1.0\r\nReply-To: reply\r\nSubject: \r\nTo: \r\nContent-Type: multipart/mixed;\r\n\tboundary=\"mixed\"; charset=UTF-8\r\n\r\n--mixed\r\nContent-Type: multipart/alternative;\r\n\tboundary=\"alt\"\r\n\r\n\r\n--alt--\r\n\r\n--mixed--\r\n",
			false,
		},
		{
			"From name",
			[]byte{},
			[]byte{},
			[]string{""},
			"",
			"",
			"name",
			"",
			"From: name <>\r\nMime-Version: 1.0\r\nSubject: \r\nTo: \r\nContent-Type: multipart/mixed;\r\n\tboundary=\"mixed\"; charset=UTF-8\r\n\r\n--mixed\r\nContent-Type: multipart/alternative;\r\n\tboundary=\"alt\"\r\n\r\n\r\n--alt--\r\n\r\n--mixed--\r\n",
			false,
		},
		{
			"From name + address",
			[]byte{},
			[]byte{},
			[]string{""},
			"",
			"addr",
			"name",
			"",
			"From: name <addr>\r\nMime-Version: 1.0\r\nSubject: \r\nTo: \r\nContent-Type: multipart/mixed;\r\n\tboundary=\"mixed\"; charset=UTF-8\r\n\r\n--mixed\r\nContent-Type: multipart/alternative;\r\n\tboundary=\"alt\"\r\n\r\n\r\n--alt--\r\n\r\n--mixed--\r\n",
			false,
		},
		{
			"From",
			[]byte{},
			[]byte{},
			[]string{""},
			"",
			"from",
			"",
			"",
			"From: from\r\nMime-Version: 1.0\r\nSubject: \r\nTo: \r\nContent-Type: multipart/mixed;\r\n\tboundary=\"mixed\"; charset=UTF-8\r\n\r\n--mixed\r\nContent-Type: multipart/alternative;\r\n\tboundary=\"alt\"\r\n\r\n\r\n--alt--\r\n\r\n--mixed--\r\n",
			false,
		},
		{
			"Subject",
			[]byte{},
			[]byte{},
			[]string{""},
			"subject",
			"",
			"",
			"",
			"From: \r\nMime-Version: 1.0\r\nSubject: subject\r\nTo: \r\nContent-Type: multipart/mixed;\r\n\tboundary=\"mixed\"; charset=UTF-8\r\n\r\n--mixed\r\nContent-Type: multipart/alternative;\r\n\tboundary=\"alt\"\r\n\r\n\r\n--alt--\r\n\r\n--mixed--\r\n",
			false,
		},
		{
			"To addresses",
			[]byte{},
			[]byte{},
			[]string{"one", "two"},
			"",
			"",
			"",
			"",
			"From: \r\nMime-Version: 1.0\r\nSubject: \r\nTo: one\r\nTo: two\r\nContent-Type: multipart/mixed;\r\n\tboundary=\"mixed\"; charset=UTF-8\r\n\r\n--mixed\r\nContent-Type: multipart/alternative;\r\n\tboundary=\"alt\"\r\n\r\n\r\n--alt--\r\n\r\n--mixed--\r\n",
			false,
		},
	}

	regex := regexp.MustCompile("\r?\n")

	for _, tt := range tests {
		m := &MailYak{
			html:      tt.rHTML,
			plain:     tt.rPlain,
			toAddrs:   tt.rtoAddrs,
			subject:   tt.rsubject,
			fromAddr:  tt.rfromAddr,
			fromName:  tt.rfromName,
			replyTo:   tt.rreplyTo,
			trimRegex: regex,
		}

		got, err := m.buildMimeWithBoundaries("mixed", "alt")
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. MailYak.buildMime() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}

		if got.String() != tt.want {
			t.Errorf("%q. MailYak.buildMime() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// TestMailYakBuildMime_withAttachments ensures attachments are correctly added to the MIME message
func TestMailYakBuildMime_withAttachments(t *testing.T) {
	tests := []struct {
		// Test description.
		name string
		// Receiver fields.
		rHTML        []byte
		rPlain       []byte
		rtoAddrs     []string
		rsubject     string
		rfromAddr    string
		rfromName    string
		rreplyTo     string
		rattachments []attachment
		// Expected results.
		wantAttach []string
		wantErr    bool
	}{
		{
			"No attachment",
			[]byte{},
			[]byte{},
			[]string{""},
			"",
			"",
			"",
			"",
			[]attachment{},
			[]string{},
			false,
		},
		{
			"One attachment",
			[]byte{},
			[]byte{},
			[]string{""},
			"",
			"",
			"",
			"",
			[]attachment{
				{"test.txt", strings.NewReader("content")},
			},
			[]string{"Y29udGVudA=="},
			false,
		},
		{
			"Two attachments",
			[]byte{},
			[]byte{},
			[]string{""},
			"",
			"",
			"",
			"",
			[]attachment{
				{"test.txt", strings.NewReader("content")},
				{"another.txt", strings.NewReader("another")},
			},
			[]string{"Y29udGVudA==", "YW5vdGhlcg=="},
			false,
		},
	}

	regex := regexp.MustCompile("\r?\n")

	for _, tt := range tests {
		m := &MailYak{
			html:        tt.rHTML,
			plain:       tt.rPlain,
			toAddrs:     tt.rtoAddrs,
			subject:     tt.rsubject,
			fromAddr:    tt.rfromAddr,
			fromName:    tt.rfromName,
			replyTo:     tt.rreplyTo,
			attachments: tt.rattachments,
			trimRegex:   regex,
		}

		got, err := m.buildMimeWithBoundaries("mixed", "alt")
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. MailYak.buildMime() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}

		seen := 0
		mr := multipart.NewReader(got, "mixed")

		// Itterate over the mime parts, look for attachments
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("%q. MailYak.buildMime() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}

			// Read the attachment data
			slurp, err := ioutil.ReadAll(p)
			if err != nil {
				t.Errorf("%q. MailYak.buildMime() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}

			// Skip non-attachments
			if p.Header.Get("Content-Disposition") == "" {
				continue
			}

			// Run through our attachments looking for a match
			for i, attch := range tt.rattachments {
				// Check Disposition header
				if p.Header.Get("Content-Disposition") != fmt.Sprintf("attachment; filename=%s", attch.filename) {
					continue
				}

				// Check data
				if !bytes.Equal(slurp, []byte(tt.wantAttach[i])) {
					fmt.Printf("Part %q: %q\n", p.Header.Get("Content-Disposition"), slurp)
					continue
				}

				seen++
			}

		}

		// Did we see all the expected attachments?
		if seen != len(tt.rattachments) {
			t.Errorf("%q. MailYak.buildMime() didn't find all attachments in mime body", tt.name)
		}
	}
}
