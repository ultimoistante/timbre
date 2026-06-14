package stream

import (
	"bytes"
	"io"
	"testing"
)

func TestParseStreamTitle(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"basic", "StreamTitle='Artist - Song';StreamUrl='http://x';", "Artist - Song"},
		{"only title", "StreamTitle='Just A Title';", "Just A Title"},
		{"empty", "StreamTitle='';", ""},
		{"no title key", "StreamUrl='http://x';", ""},
		{"nul padding", "StreamTitle='Padded';\x00\x00\x00", "Padded"},
		{"quote in middle, no terminator", "StreamTitle='Half", "Half"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := parseStreamTitle([]byte(c.in)); got != c.want {
				t.Fatalf("parseStreamTitle(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// buildICY frames audio + a metadata block the way an Icecast server does:
// metaInt audio bytes, one length byte, then lenByte*16 metadata bytes.
func buildICY(audio []byte, metaInt int, metaText string) []byte {
	var b bytes.Buffer
	b.Write(audio[:metaInt])
	// pad metadata to a multiple of 16
	meta := []byte(metaText)
	for len(meta)%16 != 0 {
		meta = append(meta, 0)
	}
	b.WriteByte(byte(len(meta) / 16))
	b.Write(meta)
	b.Write(audio[metaInt:])
	return b.Bytes()
}

func TestDemuxICY_StripsMetadataAndForwardsAudio(t *testing.T) {
	metaInt := 32
	audio := bytes.Repeat([]byte{0xAB}, metaInt*2)
	stream := buildICY(audio, metaInt, "StreamTitle='Now Playing';")

	// Append a trailing empty metadata block so the second audio chunk is
	// followed by a valid length byte (0 = no metadata), exercising that path.
	stream = append(stream, 0x00)

	var out bytes.Buffer
	var gotTitle string
	err := demuxICY(&out, bytes.NewReader(stream), metaInt, nil, func(s string) { gotTitle = s })
	if err != nil {
		t.Fatalf("demuxICY error: %v", err)
	}

	if gotTitle != "Now Playing" {
		t.Fatalf("title = %q, want %q", gotTitle, "Now Playing")
	}
	if !bytes.Equal(out.Bytes(), audio) {
		t.Fatalf("forwarded audio mismatch: got %d bytes, want %d (metadata leaked?)", out.Len(), len(audio))
	}
}

func TestExtractOGImage(t *testing.T) {
	cases := []struct {
		name string
		html string
		want string
	}{
		{"og image", `<head><meta property="og:image" content="https://x/logo.png"></head>`, "https://x/logo.png"},
		{"og image reversed attrs", `<meta content="https://x/a.jpg" property="og:image" />`, "https://x/a.jpg"},
		{"og preferred over twitter", `<meta name="twitter:image" content="https://x/t.png"><meta property="og:image" content="https://x/o.png">`, "https://x/o.png"},
		{"twitter fallback", `<meta name="twitter:image" content="https://x/t.png">`, "https://x/t.png"},
		{"none", `<meta name="description" content="hi">`, ""},
		{"single quotes", `<meta property='og:image' content='https://x/q.png'>`, "https://x/q.png"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := extractOGImage(c.html); got != c.want {
				t.Fatalf("extractOGImage = %q, want %q", got, c.want)
			}
		})
	}
}

func TestCopyFlush_Passthrough(t *testing.T) {
	in := bytes.Repeat([]byte{0x01, 0x02}, 50000)
	var out bytes.Buffer
	if err := copyFlush(&out, bytes.NewReader(in), nil); err != nil && err != io.EOF {
		t.Fatalf("copyFlush error: %v", err)
	}
	if !bytes.Equal(out.Bytes(), in) {
		t.Fatal("passthrough mismatch")
	}
}
