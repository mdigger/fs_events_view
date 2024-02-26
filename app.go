package main

import (
	"fmt"
	"strings"
	"time"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
)

type App struct {
	events       []Event
	autocomplete []string
	filter       *cview.InputField
	list         *cview.List
	text         *cview.TextView
	app          *cview.Application
}

func NewApp(events []Event, autocomplete []string, search ...string) *App {
	// фильтр для поиска
	filter := cview.NewInputField()
	filter.SetLabel("[::u]F[::-]ilter: ")     // spell-checker:disable-line
	filter.SetText(strings.Join(search, " ")) // сразу подставляем поисковый запрос
	filter.SetPadding(0, 0, 1, 1)

	// список событий
	list := cview.NewList()
	list.ShowSecondaryText(false)
	list.SetHighlightFullLine(true)
	list.SetTitle("Events")
	list.SetBorder(true)
	list.SetBorderAttributes(tcell.AttrDim)

	// описание события
	text := cview.NewTextView()
	text.SetDynamicColors(true)
	text.SetBorder(true)
	text.SetBorderAttributes(tcell.AttrDim)

	// разделение списка событий и описания
	split := cview.NewFlex()
	split.AddItem(list, 32, 0, true)
	split.AddItem(text, 0, 1, false)

	// структура окна приложения
	flex := cview.NewFlex()
	flex.SetDirection(cview.FlexRow)
	flex.AddItem(split, 0, 1, true)
	flex.AddItem(filter, 1, 0, false)

	// основное окно приложения
	app := cview.NewApplication()
	app.SetRoot(flex, true)
	app.EnableMouse(true)

	// дополнительные настройки внешнего вида

	// инициализируем данные приложения
	application := &App{
		events:       events,
		autocomplete: autocomplete,
		filter:       filter,
		list:         list,
		text:         text,
		app:          app,
	}

	list.SetChangedFunc(application.listItemSelected)
	list.SetSelectedFunc(application.listItemSelected)
	list.SetDoneFunc(application.listItemDone)
	filter.SetDoneFunc(application.filterDone)
	filter.SetAutocompleteFunc(application.autoComplete)
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
	for _, event := range a.events {
		if filter == "" || event.Contains(filter) {
			item := cview.NewListItem(event.Name)
			item.SetReference(event)
			a.list.AddItem(item)
		}
	}

	// задаём заголовок списка событий
	if c := a.list.GetItemCount(); c < len(a.events) {
		a.list.SetTitle(fmt.Sprintf("[::d]Filtered:[::-] %d/%d", c, len(a.events)))
	} else {
		a.list.SetTitle(fmt.Sprintf("[::d]Total:[::-] %d", c))
	}

	a.list.SetOffset(0, 0) // всегда поле обновления переходим к началу
	a.app.SetFocus(a.list) // переводим фокус на список событий
}

func (a *App) listItemSelected(_ int, item *cview.ListItem) {
	a.text.Clear()
	event, ok := item.GetReference().(Event)
	if !ok {
		panic("invalid event type")
	}
	if err := event.Format(a.text, a.filter.GetText()); err != nil {
		panic(err)
	}

	title := fmt.Sprintf("%s [::d]#%d [%s]",
		event.Name, event.Sequence, event.Timestamp.Format(time.TimeOnly))
	a.text.SetTitle(title)
	a.text.ScrollToBeginning()
}

func (a *App) autoComplete(currentText string) (entries []*cview.ListItem) {
	if currentText == "" {
		return nil
	}

	for _, word := range a.autocomplete {
		if strings.HasPrefix(word, currentText) {
			item := cview.NewListItem(word)
			entries = append(entries, item)
		}
	}

	return entries
}

func (a *App) filterDone(_ tcell.Key) {
	a.fillList()
}

func (a *App) listItemDone() {
	a.filter.SetText("")
	a.fillList()
}

func (a *App) setInputCapture(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyCtrlF {
		a.app.SetFocus(a.filter) // переключаемся на поле фильтра
		return nil
	}

	return event
}
