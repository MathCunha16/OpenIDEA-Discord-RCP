package main

import (
	"context"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"OpenIDEA-Discord-Rcp/internal/ide"
)

func TestReadEventsDecodesLinesAndClosesOnEOF(t *testing.T) {
	input := `{"type":"editor-focused","file":"/src/main.go","project":"/src"}
not json
{"type":"scheduled","payload":"{}"}
`
	out := make(chan *ide.EditorEvent, 8)
	logger := log.New(io.Discard, "", 0)

	readEvents(context.Background(), strings.NewReader(input), out, logger)

	var got []string
	for ev := range out {
		got = append(got, ev.Type)
	}
	if len(got) != 2 || got[0] != "editor-focused" || got[1] != "scheduled" {
		t.Fatalf("unexpected events: %v", got)
	}
}

func TestReadEventsStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	out := make(chan *ide.EditorEvent)
	done := make(chan struct{})
	go func() {
		readEvents(ctx, strings.NewReader(`{"type":"scheduled"}`+"\n"), out, log.New(io.Discard, "", 0))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("readEvents did not return after context cancellation")
	}
}
