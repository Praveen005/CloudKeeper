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
	Action string
}

// var FilesToAdd map[string]FileChangeEvent
// var FilesToRemove map[string]FileChangeEvent
var FilesToUpdate map[string]FileChangeEvent

func init() {
	// FilesToAdd = make(map[string]FileChangeEvent)
	// FilesToRemove = make(map[string]FileChangeEvent)
	FilesToUpdate = make(map[string]FileChangeEvent)
}

func watch(ctx context.Context) {
	c := make(chan notify.EventInfo, changeChannelBufferSize)

	regularEvents := make(chan notify.EventInfo, 1)
	renameEvents := make(chan notify.EventInfo, 2)
	dirToWatch := MetaCfg.backupDir

	// we have to set a recursive watch, hence adding a /...
	if dirToWatch[len(dirToWatch)-1] == '/' {
		dirToWatch += "..."
	} else {
		dirToWatch += "/..."
	}
	if err := notify.Watch(dirToWatch, c, notify.InCreate, notify.Remove, notify.Write, notify.InMovedFrom, notify.InMovedTo); err != nil {
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
			switch ei.Event() {
			case notify.InCreate, notify.Write:

				addEvent(ei, "add")
				fmt.Println("Regular file change event: ", ei.Path())

			case notify.Remove:

				addEvent(ei, "remove")
				fmt.Println("Regular file change event: ", ei.Path())
			}
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

				addEvent(ei, "remove")

				fmt.Println("File moved from: ", info.From)
			case notify.InMovedTo:
				info.To = ei.Path()

				addEvent(ei, "add")
				fmt.Println("File moved to: ", info.To)
			}

		case <-ctx.Done():
			return
		}
	}
}

func addEvent(ei notify.EventInfo, action string) {
	f := FileChangeEvent{
		Action: action,
	}

	FilesToUpdate[ei.Path()] = f

	// if action == "add"{
	// 	FilesToAdd[ei.Path()] = f
	// 	return
	// }
	// FilesToRemove[ei.Path()] = f
}

/*


Edge cases:

1. Files is present in s3, you deleted it(in local), but then added back the file with the same name at the same location, but with different content inside:
	How will this be processed?
	1. Remove the existing file
	2. Add the file again(content is diff.)
	caveat: If the content is same, we are making extra deletion & Addition, that's a trade-off we're making


*/
