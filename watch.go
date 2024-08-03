package main

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/sys/unix"

	"github.com/rjeczalik/notify"
)

const (
	changeChannelBufferSize = 1000
)

type FileChangeEvent struct {
	Action   string
}

var FilesToUpdate map[string]FileChangeEvent

func init() {
	FilesToUpdate = make(map[string]FileChangeEvent)
}

func watch(ctx context.Context) {
	c := make(chan notify.EventInfo, changeChannelBufferSize)

	regularEvents := make(chan notify.EventInfo, 1)
	renameEvents := make(chan notify.EventInfo, 2)

	if err := notify.Watch("/home/praveen/fsnotifyTest/...", c, notify.Create, notify.Remove, notify.Write, notify.InMovedFrom, notify.InMovedTo); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(c)

	go directEvents(ctx, c, regularEvents, renameEvents)
	go handleRegularEvents(ctx, regularEvents)
	go handleRenameEvents(ctx, renameEvents)

	select {}
}

func directEvents(ctx context.Context, c, regularEvents, renameEvents chan notify.EventInfo) {
	for {
		select {
		case eventInfo := <-c:
			event := eventInfo.Event()
			switch event {
			case notify.Create, notify.Remove, notify.Write:
				regularEvents <- eventInfo
			case notify.InMovedFrom, notify.InMovedTo:
				renameEvents <- eventInfo
			}
		case <-ctx.Done():
			return
		}
	}
}

func handleRegularEvents(ctx context.Context, regularEvents chan notify.EventInfo) {
	for {
		select {
		case ei := <-regularEvents:
			if _, ok := FilesToUpdate[ei.Path()]; !ok {
				addEvent(ei, true)
			}
			fmt.Println("File change event: ", ei.Event().String())
		case <-ctx.Done():
			return
		}
	}
}

func handleRenameEvents(ctx context.Context, renameEvents chan notify.EventInfo) {
	moves := make(map[uint32]struct {
		From string
		To   string
	})

	for {
		select {
		case ei := <-renameEvents:
			cookie := ei.Sys().(*unix.InotifyEvent).Cookie
			info := moves[cookie]

			switch ei.Event() {
			case notify.InMovedFrom:
				info.From = ei.Path()
				if _, ok := FilesToUpdate[info.From]; !ok {
					addEvent(ei, false)
				}

				fmt.Println("File moved from: ", info.From)
			case notify.InMovedTo:
				info.To = ei.Path()
				if _, ok := FilesToUpdate[info.To]; !ok {
					addEvent(ei, true)
				}
				fmt.Println("File moved to: ", info.To)
			}

		case <-ctx.Done():
			return
		}
	}
}

func addEvent(ei notify.EventInfo, add bool) {
	f := FileChangeEvent{}
	if add {
		f.Action = "add"
		FilesToUpdate[ei.Path()] = f
	} else {
		f.Action = "remove"
		FilesToUpdate[ei.Path()] = f
	}
}
