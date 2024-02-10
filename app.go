package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type App struct {
	events       []Event
	autocomplete []string
	filter       *tview.InputField
	list         *tview.List
	text         *tview.TextView
	app          *tview.Application
}

func NewApp(events []Event, autocomplete []string, search ...string) *App {
	// фильтр для поиска
	filter := tview.NewInputField().
		SetLabel("[::u]F[::-]ilter: ").    // spell-checker:disable-line
		SetText(strings.Join(search, " ")) // сразу подставляем поисковый запрос
	// список событий
	list := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true)
	// описание события
	text := tview.NewTextView().
		SetDynamicColors(true)
	// структура окна приложения
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(list, 32, 0, true).
			AddItem(text, 0, 1, false),
			0, 1, true).
		AddItem(filter, 1, 0, false)
	// основное окно приложения
	app := tview.NewApplication().
		SetRoot(flex, true).
		EnableMouse(true)

	// дополнительные настройки внешнего вида
	text.SetBorder(true).
		SetBorderAttributes(tcell.AttrDim)
	list.SetTitle("Events").
		SetBorder(true).
		SetBorderAttributes(tcell.AttrDim)
	filter.SetBorderPadding(0, 0, 1, 1)

	// инициализируем данные приложения
	application := &App{
		events:       events,
		autocomplete: autocomplete,
		filter:       filter,
		list:         list,
		text:         text,
		app:          app,
	}

	filter.SetDoneFunc(application.filterDone)
	filter.SetAutocompleteFunc(application.autoComplete)
	filter.SetAutocompletedFunc(application.setAutocompletedFunc)
	list.SetChangedFunc(application.listItemSelected)
	app.SetInputCapture(application.setInputCapture)

	return application
}

// Запускаем приложение.
func (a *App) Run() error {
	a.fillList() // заполняем список событий
	return a.app.Run()
}

func (a *App) fillList() {
	filter := a.filter.GetText() // текст для фильтрации

	// очищаем и заполняем список событий
	a.list.Clear()
	for i, event := range a.events {
		if filter == "" || event.Contains(filter) {
			a.list.AddItem(event.Name, strconv.Itoa(i), 0, nil)
		}
	}

	// задаём заголовок списка событий
	if c := a.list.GetItemCount(); c < len(a.events) {
		a.list.SetTitle(fmt.Sprintf("Filtered: %d/%d", c, len(a.events)))
	} else {
		a.list.SetTitle(fmt.Sprintf("Total: %d", c))
	}

	a.list.SetOffset(0, 0) // всегда поле обновления переходим к началу
	a.app.SetFocus(a.list) // переводим фокус на список событий
}

func (a *App) listItemSelected(_ int, mainText, secondaryText string, _ rune) {
	i, err := strconv.Atoi(secondaryText)
	if err != nil {
		panic(err)
	}
	event := a.events[i]

	w := a.text.BatchWriter()
	defer w.Close()
	w.Clear()
	if err := event.Format(w, a.filter.GetText()); err != nil {
		panic(err)
	}

	title := fmt.Sprintf("%s [::d]#%d [%s]",
		mainText, event.Sequence, event.Timestamp.Format(time.TimeOnly))
	a.text.SetTitle(title)
	a.text.ScrollToBeginning()
}

func (a *App) autoComplete(currentText string) (entries []string) {
	if currentText == "" {
		return nil
	}

	for _, word := range a.autocomplete {
		if strings.HasPrefix(word, currentText) {
			entries = append(entries, word)
		}
	}

	return entries
}

func (a *App) setAutocompletedFunc(text string, _, source int) bool {
	if source != tview.AutocompletedNavigate {
		a.filter.SetText(text)
	}

	return source == tview.AutocompletedEnter ||
		source == tview.AutocompletedClick
}

func (a *App) filterDone(_ tcell.Key) {
	a.fillList()
}

func (a *App) setInputCapture(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyCtrlF {
		a.app.SetFocus(a.filter) // переключаемся на поле фильтра
		return nil
	}

	return event
}
