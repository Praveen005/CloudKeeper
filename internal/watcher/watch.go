package watcher

import (
	"context"
	"fmt"
	"log"
	"sync"

	"golang.org/x/sys/unix"

	"github.com/rjeczalik/notify"

	"github.com/Praveen005/CloudKeeper/internal/db"
	"github.com/Praveen005/CloudKeeper/internal/fsconfig"
)

const (
	eventChannelBufferSize = 1000
)

// Watch function keeps an eye over the directory you want to backup for any modfication.
func Watch(ctx context.Context) {
	c := make(chan notify.EventInfo, eventChannelBufferSize)

	regularEvents := make(chan notify.EventInfo, 1) // stores events like, creation/motification/removal
	renameEvents := make(chan notify.EventInfo, 2)  // stores Rename events(In rename previous file is deleted, and a new one is created with the same name. It also caters for file/folder movement, like: moved from & moved to)
	dirToWatch := fsconfig.MetaCfg.BackupDir

	// we have to set a recursive Watch, hence adding a /...
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

	// go DirectEvents(ctx, c, regularEvents, renameEvents)
	// go HandleRegularEvents(ctx, regularEvents)
	// go HandleRenameEvents(ctx, renameEvents)

	// select {}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		DirectEvents(ctx, c, regularEvents, renameEvents)
	}()

	go func() {
		defer wg.Done()
		HandleRegularEvents(ctx, regularEvents)
	}()

	go func() {
		defer wg.Done()
		go HandleRenameEvents(ctx, renameEvents)
	}()
	wg.Wait()
}

// DirectEvents funcion consumes events from channel 'c' and directs them to appropriate channels
func DirectEvents(ctx context.Context, c, regularEvents, renameEvents chan notify.EventInfo) {
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

// HandleRegularEvents takes in events like creation/motification/removal
func HandleRegularEvents(ctx context.Context, regularEvents chan notify.EventInfo) {
	for {
		select {
		case ei := <-regularEvents:
			switch ei.Event() {
			case notify.InCreate, notify.Write:

				AddEvent(ei, "add")
				fmt.Println("Regular file change event: ", ei.Path())

			case notify.Remove:

				AddEvent(ei, "remove")
				fmt.Println("Regular file change event: ", ei.Path())
			}
		case <-ctx.Done():
			return
		}
	}
}

// HandleRenameEvents function takes in events like, Rename, and file/folder movement from one to another
func HandleRenameEvents(ctx context.Context, renameEvents chan notify.EventInfo) {
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

				AddEvent(ei, "remove")

				fmt.Println("File moved from: ", info.From)
			case notify.InMovedTo:
				info.To = ei.Path()

				AddEvent(ei, "add")
				fmt.Println("File moved to: ", info.To)
			}

		case <-ctx.Done():
			return
		}
	}
}

// AddEvent function stores the file change metadata in-memory
func AddEvent(ei notify.EventInfo, action string) {
	f := db.FileChangeEvent{
		Action: action,
	}

	db.FilesToUpdate[ei.Path()] = f
}

// Observation: DirectEvents, HandleRegularEvents & HandleRenameEvents functions can very well be clubbed together :)
