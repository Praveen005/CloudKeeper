package main

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/sys/unix"

	"github.com/rjeczalik/notify"
)

const (
	eventChannelBufferSize = 1000
)

// FileChangeEvent stores the action(add/remove) to be performed a given file path
type FileChangeEvent struct {
	Action string
}

// FilesToUpdate function stores the metadata(filepath and action to be performed on it) in-memory to be flushed later to DB
var FilesToUpdate map[string]FileChangeEvent

// Initializes the map at the start of the program
func init() {
	FilesToUpdate = make(map[string]FileChangeEvent)
}

// watch function keeps an eye over the directory you want to backup for any modfication.
func watch(ctx context.Context) {
	c := make(chan notify.EventInfo, eventChannelBufferSize)

	regularEvents := make(chan notify.EventInfo, 1) // stores events like, creation/motification/removal
	renameEvents := make(chan notify.EventInfo, 2)  // stores Rename events(In rename previous file is deleted, and a new one is created with the same name. It also caters for file/folder movement, like: moved from & moved to)
	dirToWatch := MetaCfg.backupDir

	// we have to set a recursive watch, hence adding a /...
	if dirToWatch[len(dirToWatch)-1] == '/' {
		dirToWatch += "..."
	} else {
		dirToWatch += "/..."
	}
	// As and when any event occurs, it is stored in channel 'c'. It is of the type notify.EventInfo
	if err := notify.Watch(dirToWatch, c, notify.InCreate, notify.Remove, notify.Write, notify.InMovedFrom, notify.InMovedTo); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(c)

	go directEvents(ctx, c, regularEvents, renameEvents)
	go handleRegularEvents(ctx, regularEvents)
	go handleRenameEvents(ctx, renameEvents)

	select {}
}

// directEvents funcion consumes events from channel 'c' and directs them to appropriate channels
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

// handleRegularEvents takes in events like creation/motification/removal
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

// handleRenameEvents function takes in events like, Rename, and file/folder movement from one to another
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

// addEvent function stores the file change metadata in-memory
func addEvent(ei notify.EventInfo, action string) {
	f := FileChangeEvent{
		Action: action,
	}

	FilesToUpdate[ei.Path()] = f
}

// Observation: directEvents, handleRegularEvents & handleRenameEvents functions can very well be clubbed together :)
