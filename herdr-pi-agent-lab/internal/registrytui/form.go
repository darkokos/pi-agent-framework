package registrytui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/darkokos/pi-agent-framework/pi-agent-validate"
)

type fieldKind int

const (
	kindText fieldKind = iota
	kindEnum
	kindBool
	kindPicker
	kindModelPicker
	kindPath
)

const pickerVisibleLimit = 8

type formField struct {
	label string
	help  string
	kind  fieldKind

	input textinput.Model

	options []string
	enumIdx int

	boolVal bool

	pickerMatches []string
	pickerCursor  int
	appendMode    bool
	pathBase      string

	modelOptions []validate.ModelEntry
	modelMatches []validate.ModelEntry

	resolvedModel    string
	resolvedProvider string

	visible func() bool
}

type form struct {
	title    string
	fields   []*formField
	focus    int
	errMsg   string
	notices  []string
	warnings []string
	onSubmit func(f *form) error
}

func newTextField(label, help, placeholder, value string) *formField {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = placeholder
	ti.SetValue(value)

	return &formField{label: label, help: help, kind: kindText, input: ti}
}

func newEnumField(label, help string, options []string, current string) *formField {
	idx := 0
	for i, o := range options {
		if o == current {
			idx = i
		}
	}

	return &formField{label: label, help: help, kind: kindEnum, options: options, enumIdx: idx}
}

func newBoolField(label, help string, current bool) *formField {
	return &formField{label: label, help: help, kind: kindBool, boolVal: current}
}

func (f *formField) refilterPicker() {
	full := f.text()

	already, rawQuery := "", full
	if f.appendMode {
		if idx := strings.LastIndex(full, ","); idx >= 0 {
			already, rawQuery = full[:idx], full[idx+1:]
		}

		rawQuery = strings.TrimSpace(rawQuery)
	}

	if f.pathBase != "" {
		if matches := registryPathMatches(f.pathBase, rawQuery); matches != nil {
			f.pickerMatches = matches
			if f.pickerCursor >= len(f.pickerMatches) {
				f.pickerCursor = 0
			}

			return
		}
	}

	query := strings.ToLower(rawQuery)

	used := map[string]bool{}
	if f.appendMode {
		for u := range strings.SplitSeq(already, ",") {
			if u = strings.TrimSpace(u); u != "" {
				used[strings.ToLower(u)] = true
			}
		}
	}

	f.pickerMatches = nil
	for _, o := range f.options {
		if used[strings.ToLower(o)] {
			continue
		}
		if query == "" || strings.Contains(strings.ToLower(o), query) {
			f.pickerMatches = append(f.pickerMatches, o)
		}
	}

	if f.pickerCursor >= len(f.pickerMatches) {
		f.pickerCursor = 0
	}
}

func newPickerField(label, help, placeholder string, options []string, value string, appendMode bool) *formField {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = placeholder
	ti.SetValue(value)

	f := &formField{label: label, help: help, kind: kindPicker, input: ti, options: options, appendMode: appendMode}
	f.refilterPicker()

	return f
}

func (f *formField) applyPickerChoice(chosen string) {
	if f.appendMode {
		cur := f.text()
		if idx := strings.LastIndex(cur, ","); idx >= 0 {
			cur = cur[:idx+1] + chosen
		} else {
			cur = chosen
		}

		f.input.SetValue(cur)
	} else {
		f.input.SetValue(chosen)
	}

	f.input.CursorEnd()
	f.refilterPicker()
}

func (f *formField) refilterModelPicker() {
	query := strings.ToLower(f.text())

	f.modelMatches = nil
	for _, e := range f.modelOptions {
		if query == "" || strings.Contains(strings.ToLower(e.Model+" "+e.Provider), query) {
			f.modelMatches = append(f.modelMatches, e)
		}
	}

	if f.pickerCursor >= len(f.modelMatches) {
		f.pickerCursor = 0
	}
}

func newModelPickerField(label, help string, entries []validate.ModelEntry, currentModel, currentProvider string) *formField {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "type to search, e.g. sonnet"
	ti.SetValue(currentModel)

	f := &formField{
		label: label, help: help, kind: kindModelPicker,
		input: ti, modelOptions: entries,
		resolvedModel: currentModel, resolvedProvider: currentProvider,
	}
	f.refilterModelPicker()

	return f
}

func (f *formField) applyModelChoice(e validate.ModelEntry) {
	f.resolvedModel, f.resolvedProvider = e.Model, e.Provider
	f.input.SetValue(e.Model)
	f.input.CursorEnd()
	f.refilterModelPicker()
}

func (f *formField) text() string { return strings.TrimSpace(f.input.Value()) }

func (f *formField) hasInput() bool {
	return f.kind == kindText || f.kind == kindPicker || f.kind == kindModelPicker || f.kind == kindPath
}

func (f *formField) focus() {
	if f.hasInput() {
		f.input.Focus()
	}
}

func (f *formField) blur() {
	if f.hasInput() {
		f.input.Blur()
	}
}

func newForm(title string, fields []*formField, onSubmit func(f *form) error) *form {
	fm := &form{title: title, fields: fields, onSubmit: onSubmit}
	if len(fm.fields) > 0 {
		fm.focus = -1
		fm.nextField()
	}

	return fm
}

func (f *formField) isVisible() bool {
	return f.visible == nil || f.visible()
}

func (fm *form) nextField() {
	if fm.focus >= 0 {
		fm.fields[fm.focus].blur()
	}

	for range fm.fields {
		fm.focus = (fm.focus + 1) % len(fm.fields)
		if fm.fields[fm.focus].isVisible() {
			break
		}
	}

	fm.fields[fm.focus].focus()
}

func (fm *form) prevField() {
	fm.fields[fm.focus].blur()

	for range fm.fields {
		fm.focus = (fm.focus - 1 + len(fm.fields)) % len(fm.fields)
		if fm.fields[fm.focus].isVisible() {
			break
		}
	}

	fm.fields[fm.focus].focus()
}

func (fm *form) handleKey(msg tea.KeyMsg) (tea.Cmd, bool, bool) {
	switch msg.String() {
	case "esc":
		return nil, false, true
	case "tab":
		if field := fm.fields[fm.focus]; field.kind == kindPath && field.completePath() {
			return nil, false, false
		}

		fm.nextField()

		return nil, false, false
	case "shift+tab":
		fm.prevField()

		return nil, false, false
	case "ctrl+w":
		if err := fm.onSubmit(fm); err != nil {
			fm.errMsg = err.Error()

			return nil, false, false
		}

		return nil, true, false
	}

	field := fm.fields[fm.focus]
	switch field.kind {
	case kindEnum:
		switch msg.String() {
		case "left", "h":
			field.enumIdx = (field.enumIdx - 1 + len(field.options)) % len(field.options)
		case "right", "l":
			field.enumIdx = (field.enumIdx + 1) % len(field.options)
		}
	case kindBool:
		if msg.String() == " " || msg.String() == "enter" {
			field.boolVal = !field.boolVal
		}
	case kindPicker:
		switch msg.String() {
		case "up":
			if field.pickerCursor > 0 {
				field.pickerCursor--
			}

			return nil, false, false
		case "down":
			if field.pickerCursor < len(field.pickerMatches)-1 {
				field.pickerCursor++
			}

			return nil, false, false
		case "enter":
			if len(field.pickerMatches) > 0 {
				field.applyPickerChoice(field.pickerMatches[field.pickerCursor])
			}

			return nil, false, false
		}

		var cmd tea.Cmd
		field.input, cmd = field.input.Update(msg)
		field.refilterPicker()

		return cmd, false, false
	case kindModelPicker:
		switch msg.String() {
		case "up":
			if field.pickerCursor > 0 {
				field.pickerCursor--
			}

			return nil, false, false
		case "down":
			if field.pickerCursor < len(field.modelMatches)-1 {
				field.pickerCursor++
			}

			return nil, false, false
		case "enter":
			if len(field.modelMatches) > 0 {
				field.applyModelChoice(field.modelMatches[field.pickerCursor])
			}

			return nil, false, false
		}

		field.resolvedModel, field.resolvedProvider = "", ""

		var cmd tea.Cmd
		field.input, cmd = field.input.Update(msg)
		field.refilterModelPicker()

		return cmd, false, false
	case kindPath:
		switch msg.String() {
		case "up":
			if field.pickerCursor > 0 {
				field.pickerCursor--
			}

			return nil, false, false
		case "down":
			if field.pickerCursor < len(field.pickerMatches)-1 {
				field.pickerCursor++
			}

			return nil, false, false
		case "enter":
			if len(field.pickerMatches) > 0 {
				field.applyPathChoice(field.pickerMatches[field.pickerCursor])
			}

			return nil, false, false
		}

		var cmd tea.Cmd
		field.input, cmd = field.input.Update(msg)
		field.refilterPath()

		return cmd, false, false
	case kindText:
		var cmd tea.Cmd
		field.input, cmd = field.input.Update(msg)

		return cmd, false, false
	}

	return nil, false, false
}

func boolLabel(v bool) string {
	if v {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("true")
	}

	return dimStyle.Render("false")
}

func writePickerMatches(b *strings.Builder, f *formField) {
	if len(f.pickerMatches) == 0 {
		b.WriteString("\n    ")
		b.WriteString(dimStyle.Render("no matches"))

		return
	}

	for j, o := range f.pickerMatches {
		if j >= pickerVisibleLimit {
			b.WriteString("\n    ")
			b.WriteString(dimStyle.Render(fmt.Sprintf("... %d more", len(f.pickerMatches)-pickerVisibleLimit)))

			break
		}

		b.WriteString("\n    ")
		if j == f.pickerCursor {
			b.WriteString(selectedStyle.Render("> " + o))
		} else {
			b.WriteString("  ")
			b.WriteString(o)
		}
	}
}

func writeModelMatches(b *strings.Builder, f *formField) {
	if len(f.modelMatches) == 0 {
		b.WriteString("\n    ")
		b.WriteString(dimStyle.Render("no matches"))

		return
	}

	for j, e := range f.modelMatches {
		if j >= pickerVisibleLimit {
			b.WriteString("\n    ")
			b.WriteString(dimStyle.Render(fmt.Sprintf("... %d more", len(f.modelMatches)-pickerVisibleLimit)))

			break
		}

		line := fmt.Sprintf("%s  (%s)", e.Model, e.Provider)
		b.WriteString("\n    ")
		if j == f.pickerCursor {
			b.WriteString(selectedStyle.Render("> " + line))
		} else {
			b.WriteString("  ")
			b.WriteString(line)
		}
	}
}

func (fm *form) view() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render(fm.title))
	b.WriteString("\n\n")

	for i, f := range fm.fields {
		if !f.isVisible() {
			continue
		}

		focused := i == fm.focus

		label := f.label
		if focused {
			label = selectedStyle.Render("> " + label)
		} else {
			label = "  " + label
		}

		b.WriteString(label)
		b.WriteString("\n    ")

		switch f.kind {
		case kindText:
			b.WriteString(f.input.View())
		case kindEnum:
			b.WriteString(f.options[f.enumIdx])
			b.WriteString(dimStyle.Render(" (h/l or arrows to change)"))
		case kindBool:
			b.WriteString(boolLabel(f.boolVal))
			b.WriteString(dimStyle.Render(" (space to toggle)"))
		case kindPicker:
			b.WriteString(f.input.View())
			if focused {
				b.WriteString(dimStyle.Render(" (type to filter, up/down move, enter select)"))
				writePickerMatches(&b, f)
			}
		case kindPath:
			b.WriteString(f.input.View())
			switch pathKind(f.text()) {
			case "file":
				b.WriteString(okStyle.Render(" (file)"))
			case "dir":
				b.WriteString(dimStyle.Render(" (directory)"))
			}
			if focused {
				b.WriteString(dimStyle.Render(" (tab complete, up/down move, enter select)"))
				writePickerMatches(&b, f)
			}
		case kindModelPicker:
			b.WriteString(f.input.View())
			if f.resolvedProvider != "" {
				b.WriteString(okStyle.Render(" (provider: " + f.resolvedProvider + ")"))
			} else if f.text() != "" {
				b.WriteString(errStyle.Render(" (not selected - pick a match below)"))
			}
			if focused {
				b.WriteString("\n    ")
				b.WriteString(dimStyle.Render("type to search, up/down move, enter confirm"))
				writeModelMatches(&b, f)
			}
		}

		if f.help != "" {
			b.WriteString("\n    ")
			b.WriteString(dimStyle.Render(f.help))
		}

		b.WriteString("\n\n")
	}

	if fm.errMsg != "" {
		b.WriteString(errStyle.Render("error: " + fm.errMsg))
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyle.Render("tab/shift+tab move  ctrl+w save  esc cancel"))

	return b.String()
}
