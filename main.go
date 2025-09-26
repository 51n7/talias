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

func flattenOptions(options []Option) []Option {
	var result []Option
	for _, opt := range options {
		if len(opt.Children) > 0 {
			// Only add children, skip the parent
			result = append(result, flattenOptions(opt.Children)...)
		} else {
			// Add leaf nodes (items with commands)
			result = append(result, opt)
		}
	}
	return result
}

func fuzzySearch(query string, options []Option) []Option {
	if query == "" {
		return options
	}
	
	var results []Option
	queryLower := strings.ToLower(query)
	
	for _, opt := range options {
		titleLower := strings.ToLower(opt.Title)
		
		// Check if query matches title or details
		if strings.Contains(titleLower, queryLower) {
			results = append(results, opt)
		}
	}
	
	return results
}

// expands ~/ to the user's home directory
func expandCommand(command string) string {
	if !strings.Contains(command, "~/") {
		return command
	}
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return command // Return original command if home dir can't be determined
	}
	
	return strings.ReplaceAll(command, "~/", filepath.Join(homeDir, "")+"/")
}

// execute command and stop the app
func executeCommand(option Option, app *tview.Application) {
	if len(option.Command) == 0 {
		return
	}
	
	expandedCommand := expandCommand(option.Command)
	fmt.Print(expandedCommand)
	app.Stop()
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
	
	// Search state
	var searchMode bool = false
	var searchQuery string = ""
	var searchResults []Option
	var allOptions []Option = flattenOptions(rootOptions) // Flattened list of all options for search

	// Top: list
	list := tview.NewList()
	list.SetBackgroundColor(tcell.ColorDefault)

	// Search input field
	searchInput := tview.NewInputField().
		SetLabel("Search: ").
		SetFieldWidth(50)
	searchInput.SetBackgroundColor(tcell.ColorDefault)
	
	// Custom input capture for search input to handle up/down navigation
	searchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle up/down arrow keys for list navigation without changing focus
		if event.Key() == tcell.KeyUp {
			currentIndex := list.GetCurrentItem()
			if currentIndex > 0 {
				list.SetCurrentItem(currentIndex - 1)
			}
			return nil
		}
		if event.Key() == tcell.KeyDown {
			currentIndex := list.GetCurrentItem()
			if currentIndex < len(searchResults)-1 {
				list.SetCurrentItem(currentIndex + 1)
			}
			return nil
		}
		// Handle Enter to execute selected command
		if event.Key() == tcell.KeyEnter {
			if len(searchResults) > 0 && list.GetCurrentItem() >= 0 {
				selectedIndex := list.GetCurrentItem()
				if selectedIndex < len(searchResults) {
					executeCommand(searchResults[selectedIndex], app)
				}
			}
			return nil
		}
		// Let all other keys (including left/right arrows) pass through to the input field
		return event
	})

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
					executeCommand(option, app)
				}
			})
		}
	}

	// Function to populate search results
	var populateSearchResults func()
	populateSearchResults = func() {
		list.Clear()
		searchResults = fuzzySearch(searchQuery, allOptions)
		for _, option := range searchResults {
			opt := option // capture
			displayTitle := opt.Title
			if len(opt.Children) > 0 {
				displayTitle = "> " + opt.Title
			}
			list.AddItem(displayTitle, "", 0, func() {
				executeCommand(opt, app)
			})
		}
	}

	// Initial population
	populateList()

	// Update bottom panel when selection changes
	list.SetChangedFunc(func(index int, mainText string, _ string, _ rune) {
		if searchMode {
			if index >= 0 && index < len(searchResults) {
				infoBox.SetText(searchResults[index].Details)
			}
		} else {
			if index >= 0 && index < len(currentOptions) {
				infoBox.SetText(currentOptions[index].Details)
			}
		}
	})

	// Search input change handler
	searchInput.SetChangedFunc(func(text string) {
		searchQuery = text
		populateSearchResults()
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

	// Function to switch to search mode
	switchToSearchMode := func() {
		searchMode = true
		searchQuery = ""
		searchInput.SetText("")
		grid.Clear().
			SetRows(1, 0, 5).
			SetColumns(0).
			SetBorders(true).
			SetBordersColor(tcell.ColorWhite).
			AddItem(searchInput, 0, 0, 1, 1, 0, 0, true).
			AddItem(list, 1, 0, 1, 1, 0, 0, false).
			AddItem(infoBox, 2, 0, 1, 1, 0, 0, false)
		app.SetFocus(searchInput)
		populateSearchResults()
		infoBox.SetText("Search mode - type to filter options")
		
		// Add custom input capture for list in search mode
		list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			// If user presses any printable character, return focus to search input
			if event.Key() == tcell.KeyRune {
				app.SetFocus(searchInput)
				// Append the character to the search input
				currentText := searchInput.GetText()
				searchInput.SetText(currentText + string(event.Rune()))
				return nil
			}
			// Handle Enter to execute command
			if event.Key() == tcell.KeyEnter {
				if len(searchResults) > 0 && list.GetCurrentItem() >= 0 {
					selectedIndex := list.GetCurrentItem()
					if selectedIndex < len(searchResults) {
						executeCommand(searchResults[selectedIndex], app)
					}
				}
				return nil
			}
			return event
		})
	}

	// Function to switch back to normal mode
	switchToNormalMode := func() {
		searchMode = false
		searchQuery = ""
		// Clear the list input capture to restore normal behavior
		list.SetInputCapture(nil)
		grid.Clear().
			SetRows(0, 5).
			SetColumns(0).
			SetBorders(true).
			SetBordersColor(tcell.ColorWhite).
			AddItem(list, 0, 0, 1, 1, 0, 0, true).
			AddItem(infoBox, 1, 0, 1, 1, 0, 0, false)
		app.SetFocus(list)
		populateList()
		infoBox.SetText("Select an option from " + currentTitle)
	}

	// Global input capture for navigation and quit
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		
		// 'q' always quits the application
		if event.Key() == tcell.KeyRune && event.Rune() == 'q' {
			app.Stop()
			return nil
		}
		// '?' enters search mode
		if event.Key() == tcell.KeyRune && event.Rune() == '?' {
			switchToSearchMode()
			return nil
		}
		// Escape: go back if in submenu, quit if at top level, exit search if in search mode
		if event.Key() == tcell.KeyEscape {
			if searchMode {
				switchToNormalMode()
			} else if len(menuStack) > 0 {
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
