package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
)

// Event represents an ESL event with headers and a body.
type Event struct {
	Name      string
	Sequence  int64
	Timestamp time.Time
	Header    string
	Body      string
}

func NewEvent(event map[string]string) Event {
	// имя события (для CUSTOM используется имя подкласса)
	name, ok := event["Event-Subclass"]
	if !ok {
		name = event["Event-Name"]
	}

	// порядковый номер события
	sequence, _ := strconv.ParseInt(event["Event-Sequence"], 10, 64)

	// время создания события
	var timestamp time.Time
	if i, err := strconv.ParseInt(event["Event-Date-Timestamp"], 10, 64); err == nil {
		timestamp = time.UnixMicro(i)
	}

	// по договоренности тело сообщения в JSON передается с таким именем
	body, ok := event["_body"]
	if ok {
		delete(event, "_body")
	}

	// сортируем порядок следования заголовков
	keys := make([]string, 0, len(event))

	for key := range event {
		// игнорируем часто используемые дополнительные заголовки
		if _, ok := ignoreHeaders[key]; ok || key == "Content-Length" {
			continue
		}

		keys = append(keys, key)
	}

	slices.Sort(keys)

	// формируем строку с заголовком события
	var w strings.Builder
	//nolint:errcheck // writing to buffer
	for _, k := range keys {
		w.WriteString(k)
		w.WriteString(": ")
		// удаляем пробелы и новые строки в начале и конце строки
		// значения заголовка — часто встречается для строк с переносами
		val := strings.TrimSpace(event[k])
		replaceNewLine.WriteString(&w, val)
		w.WriteByte('\n')
	}

	return Event{
		Name:      name,
		Sequence:  sequence,
		Timestamp: timestamp,
		Header:    w.String(),
		Body:      body,
	}
}

//nolint:gochecknoglobals // используется много раз после инициализации без изменений
var replaceNewLine = strings.NewReplacer("\n", "\n\t")

//nolint:gochecknoglobals // предопределенные заголовки для игнорирования
var ignoreHeaders = map[string]struct{}{
	"Core-UUID":                 {},
	"Event-Calling-File":        {},
	"Event-Calling-Function":    {},
	"Event-Calling-Line-Number": {},
	"Event-Date-GMT":            {},
	"Event-Date-Timestamp":      {},
	"FreeSWITCH-Hostname":       {},
	"FreeSWITCH-IPv4":           {},
	"FreeSWITCH-IPv6":           {},
	"FreeSWITCH-Switchname":     {}, // spellchecker:ignore Switchname
	"FreeSWITCH-Version":        {},
}

func (e *Event) ContainsRE(key *regexp.Regexp) bool {
	return key.MatchString(e.Header) ||
		key.MatchString(e.Body)
}

func color(c tcell.Color) string {
	return fmt.Sprintf("[%s]", c.TrueColor().String())
}

//nolint:gochecknoglobals // предопределенные цвета
var (
	selected    = color(cview.Styles.SecondaryTextColor)
	headerTitle = color(cview.Styles.ContrastSecondaryTextColor)
	numbers     = color(cview.Styles.TertiaryTextColor)

	reNumbers = regexp.MustCompile(`^\s*\d+\.*\d*\s*$`)
)

const clearLine = "[-:-:-]"

func (e *Event) FormatRE(w io.Writer, key *regexp.Regexp) error {
	buf := bufio.NewWriter(w)

	// вспомогательная функция для выделения цветом найденного
	//nolint:errcheck // writing to buffer
	filterFormat := func(line []byte) bool {
		if key == nil {
			return false
		}

		loc := key.FindIndex(line)
		if len(loc) != 2 {
			return false
		}

		buf.WriteString(selected)
		buf.Write(line[:loc[0]])
		buf.WriteString("[::rb]")
		buf.Write(line[loc[0]:loc[1]])
		buf.WriteString("[::-]")
		buf.Write(line[loc[1]:])
		buf.WriteString(clearLine) // сбрасываем форматирование

		return true
	}

	// заголовок события
	scanner := bufio.NewScanner(strings.NewReader(e.Header))
	firstLine := true
	//nolint:errcheck // writing to buffer
	for scanner.Scan() {
		line := scanner.Bytes()

		if firstLine {
			firstLine = false
		} else {
			buf.WriteByte('\n')
		}

		if filterFormat(line) {
			continue // строка уже отформатирована
		}

		// игнорируем форматирование для строк с отступом
		if len(line) > 0 && line[0] == '\t' {
			buf.Write(line)
			// buf.WriteByte('\n')

			continue
		}

		// разделяем цветом ключ заголовка и значение
		key, value, _ := bytes.Cut(line, []byte(": "))

		buf.WriteString(headerTitle)
		buf.Write(key)
		buf.WriteByte(':')

		if reNumbers.Match(value) {
			buf.WriteString(numbers) // цифровое значение
		} else {
			buf.WriteString("[-]") // не цифровое значение
		}

		buf.WriteByte(' ')
		buf.Write(value)
		buf.WriteString(clearLine) // сбрасываем форматирование
	}

	// тело события
	//nolint:errcheck // writing to buffer
	if e.Body != "" {
		// добавляем заголовок с размером тела события
		buf.WriteString(headerTitle)
		buf.WriteString("Content-Length:")
		buf.WriteString(numbers)
		buf.WriteString(strconv.Itoa(len(e.Body)))
		buf.WriteString(clearLine) // сбрасываем форматирование

		buf.WriteByte('\n') // разделитель на тела события
		if !filterFormat([]byte(e.Body)) {
			buf.WriteString(e.Body) // тело события как есть
		}
	}

	return buf.Flush() //nolint:wrapcheck
}
