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
  Title    string   `json:"title"`
  Details  string   `json:"details"`
  Command  string   `json:"command"`
  Children []Option `json:"children,omitempty"`
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

func containsOption(options []Option, target []Option) bool {
	if len(options) != len(target) {
		return false
	}
	for i, opt := range options {
		if opt.Title != target[i].Title {
			return false
		}
	}
	return true
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
	rootOptions, err := loadOptionsFromFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading options: %v\n", err)
		os.Exit(1)
	}

	// Navigation state
	var currentOptions []Option = rootOptions
	var menuStack [][]Option
	var currentTitle string = "Main Menu"

	// Top: list
	list := tview.NewList()
	list.SetBackgroundColor(tcell.ColorDefault)

	// Bottom: info box
	infoBox := tview.NewTextView().
		SetText("Welcome! Select an option.").
		SetDynamicColors(true).
		SetWrap(true)
	infoBox.SetBackgroundColor(tcell.ColorDefault)

	// Function to populate list with current options
	var populateList func()
	populateList = func() {
		list.Clear()
		for _, o := range currentOptions {
			option := o // capture
			
			// Add > prefix for items with children
			displayTitle := option.Title
			if len(option.Children) > 0 {
				displayTitle = "> " + option.Title
			}
			list.AddItem(displayTitle, "", 0, func() {

				// Check if this option has children
				if len(option.Children) > 0 {

					// Navigate to child menu
					menuStack = append(menuStack, currentOptions)
					currentOptions = option.Children
					currentTitle = option.Title
					populateList()
					infoBox.SetText("Select an option from " + currentTitle)
				} else {

					// Execute command
					expandedCommand := option.Command
					if strings.Contains(option.Command, "~/") {
						homeDir, err := os.UserHomeDir()
						if err == nil {
							expandedCommand = strings.ReplaceAll(option.Command, "~/", filepath.Join(homeDir, "")+"/")
						}
					}
					
					fmt.Print(expandedCommand)
					app.Stop()
				}
			})
		}
	}

	// Initial population
	populateList()

	// Update bottom panel when selection changes
	list.SetChangedFunc(func(index int, mainText string, _ string, _ rune) {
		if index >= 0 && index < len(currentOptions) {
			infoBox.SetText(currentOptions[index].Details)
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

	// Global input capture for navigation and quit
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		
		// 'q' always quits the application
		if event.Key() == tcell.KeyRune && event.Rune() == 'q' {
			app.Stop()
			return nil
		}

		// Escape: go back if in submenu, quit if at top level
		if event.Key() == tcell.KeyEscape {
			if len(menuStack) > 0 {

				// Go back to previous menu
				currentOptions = menuStack[len(menuStack)-1]
				menuStack = menuStack[:len(menuStack)-1]
				if len(menuStack) == 0 {
					currentTitle = "Main Menu"
				} else {

					// Find the title of the parent menu
					currentTitle = "Main Menu" // fallback
					for _, opt := range rootOptions {
						if containsOption(opt.Children, currentOptions) {
							currentTitle = opt.Title
							break
						}
					}
				}
				populateList()
				infoBox.SetText("Select an option from " + currentTitle)
			} else {
				// At top level, quit the application
				app.Stop()
			}
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
