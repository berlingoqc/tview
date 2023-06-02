// Demo code for the DeepList primitive.
package main

import (
	"log"
	"os"

	"github.com/rivo/tview"
)

// Allow selection on subitem
// Allow hightlight on subitem
// Fix movement when subitem are present

func main() {

	f, err := os.OpenFile("deep_list.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	app := tview.NewApplication()
	list := tview.NewDeepList().
		AddItem("List item 1", "", 'a', nil).
		//AddSubItem("Sub me", "", 'g', true, nil).
		//AddSubItem("Test me", "", 'g', true, nil).
		AddItem("List item 2", "", 'b', nil).
		AddItem("List item 3", "", 'c', nil)
	list.
		AddItem("List item 4", "", 'd', func() {
			list.ToggleSubListDisplay(3)
		}).
		AddSubItem("Sub me", "", 'g', false, nil).
		AddSubItem("Test me", "", 'g', false, nil).
		AddSubItem("Roll me", "", 'g', true, nil).
		AddItem("Quit", "Press to exit", 'q', func() {
			app.Stop()
		})

	if err := app.SetRoot(list, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}

}
