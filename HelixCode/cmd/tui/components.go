package main

import (
	"fmt"

	"github.com/rivo/tview"
)

// UIComponents contains reusable UI components for the Terminal UI
type UIComponents struct {
	tui *TerminalUI
}

// NewUIComponents creates a new UI components instance
func NewUIComponents(tui *TerminalUI) *UIComponents {
	return &UIComponents{
		tui: tui,
	}
}

// CreateForm creates a reusable form component
func (uc *UIComponents) CreateForm(title string, fields []FormField) *tview.Form {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(title)
	form.SetTitleAlign(tview.AlignLeft)

	// Add basic text input for now
	form.AddInputField("Input", "", 20, nil, nil)
	form.AddButton("Submit", nil)

	return form
}

// CreateList creates a reusable list component
func (uc *UIComponents) CreateList(title string, items []ListItem) *tview.List {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(title)
	list.SetTitleAlign(tview.AlignLeft)

	for _, item := range items {
		list.AddItem(item.MainText, item.SecondaryText, item.Shortcut, item.OnSelect)
	}

	return list
}

// CreateTable creates a reusable table component
func (uc *UIComponents) CreateTable(title string, headers []string, data [][]string) *tview.Table {
	table := tview.NewTable()
	table.SetBorder(true)
	table.SetTitle(title)
	table.SetTitleAlign(tview.AlignLeft)

	// Add headers
	for col, header := range headers {
		table.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tview.Styles.SecondaryTextColor).
			SetSelectable(false))
	}

	// Add data
	for row, rowData := range data {
		for col, cellData := range rowData {
			table.SetCell(row+1, col, tview.NewTableCell(cellData))
		}
	}

	return table
}

// CreateProgressBar creates a progress bar component
func (uc *UIComponents) CreateProgressBar(title string, current, total int) *tview.TextView {
	progress := tview.NewTextView()
	progress.SetBorder(true)
	progress.SetTitle(title)
	progress.SetTitleAlign(tview.AlignLeft)
	progress.SetDynamicColors(true)

	percentage := float64(current) / float64(total) * 100
	progress.SetText(uc.formatProgressBar(current, total, percentage))

	return progress
}

// CreateModal creates a modal dialog
func (uc *UIComponents) CreateModal(title, message string, buttons []ModalButton) *tview.Modal {
	modal := tview.NewModal()
	modal.SetText(message)
	modal.SetBackgroundColor(tview.Styles.ContrastBackgroundColor)

	buttonLabels := make([]string, len(buttons))
	for i, button := range buttons {
		buttonLabels[i] = button.Label
	}
	modal.AddButtons(buttonLabels)

	// Handle button clicks
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonIndex >= 0 && buttonIndex < len(buttons) {
			buttons[buttonIndex].OnClick()
		}
	})

	return modal
}

// CreateStatusBar creates a status bar component
func (uc *UIComponents) CreateStatusBar() *tview.TextView {
	status := tview.NewTextView()
	status.SetDynamicColors(true)
	status.SetTextAlign(tview.AlignCenter)
	status.SetText("[green]Ready")

	status.SetBorder(true)
	status.SetTitle("Status")
	return status
}

// CreateLogView creates a log/scrollable text view
func (uc *UIComponents) CreateLogView(title string) *tview.TextView {
	logView := tview.NewTextView()
	logView.SetBorder(true)
	logView.SetTitle(title)
	logView.SetTitleAlign(tview.AlignLeft)
	logView.SetDynamicColors(true)
	logView.SetScrollable(true)
	logView.SetWrap(true)

	return logView
}

// Helper functions

// formatProgressBar formats a progress bar string
func (uc *UIComponents) formatProgressBar(current, total int, percentage float64) string {
	width := 20
	filled := int(float64(width) * percentage / 100)

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	return fmt.Sprintf("[green]%s[white] %d/%d (%.1f%%)", bar, current, total, percentage)
}

// FormField represents a form field configuration
type FormField struct {
	Type          string
	Label         string
	DefaultValue  string
	Width         int
	BoolValue     bool
	Options       []string
	SelectedIndex int
	OnChange      interface{}
	OnClick       func()
}

// ListItem represents a list item configuration
type ListItem struct {
	MainText      string
	SecondaryText string
	Shortcut      rune
	OnSelect      func()
}

// ModalButton represents a modal button configuration
type ModalButton struct {
	Label   string
	OnClick func()
}
