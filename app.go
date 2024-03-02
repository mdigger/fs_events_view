package main

import (
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"
	"strings"
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
	// переопределяем цветовую палитру по умолчанию
	cview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault.TrueColor()
	cview.Styles.TitleColor = tcell.ColorLightSlateGray.TrueColor()
	cview.Styles.BorderColor = tcell.ColorSlateGray.TrueColor()

	// фильтр для поиска
	search := cview.NewInputField()
	search.SetLabel("[::u]F[::-]ind: ") // spell-checker:disable-line
	search.SetText(searchFilter)        // сразу подставляем поисковый запрос
	search.SetPlaceholder("regexp search filter...")
	search.SetFieldNoteTextColor(
		cview.Styles.ContrastSecondaryTextColor)

	// список событий
	list := cview.NewList()
	list.ShowSecondaryText(false)
	list.SetHighlightFullLine(true)
	list.SetTitle("Events")
	list.SetBorder(true)
	list.SetSelectedBackgroundColor(
		cview.Styles.MoreContrastBackgroundColor.TrueColor())
	list.SetSelectedTextColor(
		cview.Styles.PrimaryTextColor.TrueColor())

	// описание события
	text := cview.NewTextView()
	text.SetDynamicColors(true)
	text.SetBorder(true)

	// разделение списка событий и описания
	mainSplit := cview.NewFlex()
	mainSplit.AddItem(list, 32, 0, true)
	mainSplit.AddItem(text, 0, 1, false)

	// структура окна приложения
	flex := cview.NewFlex()
	flex.SetDirection(cview.FlexRow)
	flex.AddItem(mainSplit, 0, 1, true)
	flex.AddItem(search, 2, 0, false)

	frame := cview.NewFrame(flex)
	frame.SetBorders(0, 0, 0, 0, 0, 0)
	frame.AddText("[::d]-=[::-] [::b]ESL Events Viewer[::-] [::d]=-",
		true, cview.AlignCenter, cview.Styles.PrimaryTextColor)
	if version != "" {
		frame.AddText("[::d]v"+version,
			true, cview.AlignRight, cview.Styles.TertiaryTextColor)
	}
	frame.AddText(
		"[::r]^C[::-] exit [::d]|[::-] "+
			"[::r]^H[::-] hide event [::d]|[::-] "+
			"[::r]^R[::-] restore all [::d]|[::-] "+
			"[::r]^F[::-] search",
		false, cview.AlignCenter, cview.Styles.TertiaryTextColor)

	// основное окно приложения
	app := cview.NewApplication()
	app.SetRoot(frame, true)
	app.EnableMouse(true)

	// список названий событий для скрытия
	hideEventsMap := make(map[string]struct{}, len(hideEvents))
	for _, name := range hideEvents {
		if name = strings.TrimSpace(name); name != "" {
			hideEventsMap[name] = struct{}{}
		}
	}

	// инициализируем данные приложения
	application := &App{
		events: events,
		hide:   hideEventsMap,
		re:     nil,
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

	list.AddContextItem("[::u]H[::-]ide event", 'h', application.hideEvent) // spell-checker:disable-line
	list.AddContextItem("", 0, nil)
	list.AddContextItem("[::u]R[::-]estore all event", 'r', application.listReset) // spell-checker:disable-line

	contextMenu := list.ContextMenu.ContextMenuList()
	contextMenu.SetHighlightFullLine(true)
	contextMenu.SetSelectedBackgroundColor(
		cview.Styles.MoreContrastBackgroundColor.TrueColor())
	contextMenu.SetSelectedTextColor(
		cview.Styles.PrimaryTextColor.TrueColor())
	contextMenu.SetBorderColorFocused( // разделитель рисуется цветом основного текста
		cview.Styles.PrimaryTextColor.TrueColor())

	return application
}

// Запускаем приложение.
func (a *App) Run() error {
	defer a.app.HandlePanic()
	a.fillList() // заполняем список событий

	err := a.app.Run()

	// формируем команду запуска приложения с текущими параметрами
	var cmd strings.Builder
	if search := strings.TrimSpace(a.filter.GetText()); search != "" {
		cmd.WriteString(" -search=")
		cmd.WriteString(search)
	}
	if len(a.hide) > 0 {
		names := make([]string, 0, len(a.hide))
		for name := range a.hide {
			names = append(names, name)
		}

		cmd.WriteString(" -hide-events=\"")
		cmd.WriteString(strings.Join(names, ","))
		cmd.WriteByte('"')
	}
	if len(ignoreHeaders) == 0 {
		cmd.WriteString(" -all-headers")
	}
	if cmd.Len() > 0 {
		fmt.Printf("params:%s\n", cmd.String())
	}

	return err
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
	switch event.Key() {
	case tcell.KeyCtrlF: // переключаемся на поле фильтра
		a.app.SetFocus(a.filter)
	case tcell.KeyCtrlH: // скрываем событие из списка
		a.hideEvent(a.list.GetCurrentItemIndex())
	case tcell.KeyCtrlA: // восстанавливаем все события в списке
		a.listReset(0)
	default:
		return event
	}

	return nil
}

func (a *App) hideEvent(index int) {
	if item := a.list.GetItem(index); item != nil {
		a.hide[item.GetMainText()] = struct{}{}
		a.fillList()
	}
}

func (a *App) listReset(_ int) {
	a.hide = make(map[string]struct{})
	a.fillList()
}
