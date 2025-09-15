package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Option struct {
  Title   string `json:"title"`
  Details string `json:"details"`
  Command string `json:"command"`
}

func loadOptionsFromFile(filename string) ([]Option, error) {
	// Read the file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", filename, err)
	}

	// Parse JSON
	var options []Option
	err = json.Unmarshal(data, &options)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return options, nil
}

func main() {
	app := tview.NewApplication()

	// Load options from file in ~/.talias directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}
	
	configPath := filepath.Join(homeDir, ".talias", "options.json")
	options, err := loadOptionsFromFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading options: %v\n", err)
		os.Exit(1)
	}

	// Top: list
	list := tview.NewList()
	for _, o := range options {
    option := o // capture
    list.AddItem(option.Title, "", 0, func() {

      // Expand tilde to home directory in the command
      expandedCommand := option.Command
      if strings.Contains(option.Command, "~/") {
        homeDir, err := os.UserHomeDir()
        if err == nil {
          expandedCommand = strings.ReplaceAll(option.Command, "~/", filepath.Join(homeDir, "")+"/")
        }
      }
      
      fmt.Print(expandedCommand)
      app.Stop()
    })
	}
	list.SetBackgroundColor(tcell.ColorDefault)

	// Bottom: info box
	infoBox := tview.NewTextView().
		SetText("Welcome! Select an option.").
		SetDynamicColors(true).
		SetWrap(true)
	infoBox.SetBackgroundColor(tcell.ColorDefault)

	// Update bottom panel when selection changes
	list.SetChangedFunc(func(index int, mainText string, _ string, _ rune) {
		if index >= 0 && index < len(options) {
			infoBox.SetText(options[index].Details)
		}
	})

	// Grid layout
	grid := tview.NewGrid().
		SetRows(0, 5).
		SetColumns(0).
		SetBorders(true).
		SetBordersColor(tcell.ColorWhite).
		AddItem(list, 0, 0, 1, 1, 0, 0, true).
		AddItem(infoBox, 1, 0, 1, 1, 0, 0, false)
	
	grid.SetBackgroundColor(tcell.ColorDefault)

	// Global quit binding (press "q" or Escape anywhere to quit)
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if (event.Key() == tcell.KeyRune && event.Rune() == 'q') || event.Key() == tcell.KeyEscape {
			app.Stop()
			return nil
		}
		return event
	})

	// Respect terminal background
	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Clear()
		return false
	})

	if err := app.SetRoot(grid, true).SetFocus(list).Run(); err != nil {
		panic(err)
	}
}
