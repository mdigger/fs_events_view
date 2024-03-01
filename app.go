package main

import (
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"
	"time"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
)

type App struct {
	events []Event
	hide   map[string]struct{}
	re     *regexp.Regexp
	filter *cview.InputField
	list   *cview.List
	text   *cview.TextView
	app    *cview.Application
}

func NewApp(events []Event, searchFilter string, hideEvents ...string) *App {
	cview.Styles.TitleColor = tcell.ColorLightSlateGray.TrueColor()
	cview.Styles.BorderColor = tcell.ColorSlateGray.TrueColor()
	cview.Styles.BorderColor = tcell.ColorDarkSlateGray.TrueColor()
	cview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault.TrueColor()

	// фильтр для поиска
	search := cview.NewInputField()
	search.SetLabel("[::u]F[::-]ind: ") // spell-checker:disable-line
	search.SetPlaceholder("regexp search filter...")
	search.SetText(searchFilter) // сразу подставляем поисковый запрос
	search.SetFieldNoteTextColor(cview.Styles.ContrastSecondaryTextColor)
	// search.SetFieldWidth(26)
	// search.SetPadding(0, 0, 1, 1)

	// список событий
	list := cview.NewList()
	list.ShowSecondaryText(false)
	list.SetHighlightFullLine(true)
	list.SetTitle("Events")
	list.SetBorder(true)
	list.SetSelectedBackgroundColor(cview.Styles.MoreContrastBackgroundColor.TrueColor())

	// описание события
	text := cview.NewTextView()
	text.SetDynamicColors(true)
	text.SetBorder(true)
	// text.SetWrap(false)

	// разделение списка событий и описания
	mainSplit := cview.NewFlex()
	mainSplit.AddItem(list, 32, 0, true)
	mainSplit.AddItem(text, 0, 1, false)

	// структура окна приложения
	flex := cview.NewFlex()
	flex.SetDirection(cview.FlexRow)
	// flex.AddItem(header, 1, 0, false)
	flex.AddItem(mainSplit, 0, 1, true)
	flex.AddItem(search, 2, 0, false)

	frame := cview.NewFrame(flex)
	frame.SetBorders(0, 0, 0, 0, 0, 0)
	frame.AddText("[::d]-=[::-] [::b]ESL Events Viewer[::-] [::d]=-",
		true, cview.AlignCenter, cview.Styles.PrimaryTextColor)
	if version != "" {
		frame.AddText(fmt.Sprintf("[::d]v%s", version),
			true, cview.AlignRight, cview.Styles.TertiaryTextColor)
	}

	// основное окно приложения
	app := cview.NewApplication()
	app.SetRoot(frame, true)
	app.EnableMouse(true)

	// дополнительные настройки внешнего вида

	// список названий событий для скрытия
	hideEventsMap := make(map[string]struct{}, len(hideEvents))
	for _, name := range hideEvents {
		hideEventsMap[name] = struct{}{}
	}

	// инициализируем данные приложения
	application := &App{
		events: events,
		hide:   hideEventsMap,
		filter: search,
		list:   list,
		text:   text,
		app:    app,
	}

	list.SetChangedFunc(application.listItemSelected)
	list.SetSelectedFunc(application.listItemSelected)
	list.SetDoneFunc(application.listItemDone)
	search.SetDoneFunc(application.filterDone)
	app.SetInputCapture(application.setInputCapture)

	return application
}

// Запускаем приложение.
func (a *App) Run() error {
	defer a.app.HandlePanic()
	a.fillList() // заполняем список событий

	return a.app.Run()
}

func (a *App) fillList() {
	a.filter.ResetFieldNote()

	if reStr := a.filter.GetText(); reStr != "" {
		var err error
		a.re, err = regexp.Compile(`(?mi)` + reStr)
		if err != nil {
			msg := err.Error()
			var reError *syntax.Error
			if errors.As(err, &reError) {
				msg = reError.Code.String()
			}

			a.filter.SetFieldNote(msg)
		}
	} else {
		a.re = nil
	}

	// очищаем и заполняем список событий
	a.list.Clear()
	a.text.Clear()
	a.text.SetTitle("")

	for _, event := range a.events {
		// пропускаем события, которые должны быть скрыты
		if _, ok := a.hide[event.Name]; ok {
			continue
		}

		// пропускаем события, которые не содержат текст для поиска, если он определен
		if a.re == nil || event.ContainsRE(a.re) {
			item := cview.NewListItem(event.Name)
			item.SetReference(event)

			a.list.AddItem(item)
		}
	}

	// for i, item := range a.list.GetItems() {
	// 	_, ok := a.hide[item.GetMainText()]
	// 	a.list.SetItemEnabled(i, !ok)
	// }

	// задаём заголовок списка событий
	if c := a.list.GetItemCount(); c < len(a.events) {
		a.list.SetTitle(fmt.Sprintf("Filtered: [::d]%d/%d", c, len(a.events)))
	} else {
		a.list.SetTitle(fmt.Sprintf("Total: [::d]%d", c))
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

	if err := event.FormatRE(a.text, a.re); err != nil {
		panic(err)
	}

	a.text.ScrollToBeginning()

	title := fmt.Sprintf("%s [::d]#%d [%s]",
		event.Name, event.Sequence, event.Timestamp.Format(time.TimeOnly))
	a.text.SetTitle(title)
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
