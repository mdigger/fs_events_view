package main

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
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

// Contains возвращает true, если ключ содержится в заголовке или теле события.
func (e *Event) Contains(key string) bool {
	return strings.Contains(e.Header, key) ||
		strings.Contains(e.Body, key)
}

func (e *Event) Format(w io.Writer, key string) error {
	buf := bufio.NewWriter(w)

	const clearLine = "[-:-:-]\n"

	// вспомогательная функция для выделения цветом найденного
	filter := []byte(key)
	//nolint:errcheck // writing to buffer
	filterFormat := func(line []byte) bool {
		if key == "" {
			return false
		}

		before, after, ok := bytes.Cut(line, filter)
		if !ok {
			return false
		}

		buf.WriteString("[yellow]")
		buf.Write(before)
		buf.WriteString("[::rb]")
		buf.Write(filter)
		buf.WriteString("[::-]")
		buf.Write(after)
		buf.WriteString(clearLine) // сбрасываем форматирование

		return true
	}

	// заголовок события
	scanner := bufio.NewScanner(strings.NewReader(e.Header))
	//nolint:errcheck // writing to buffer
	for scanner.Scan() {
		line := scanner.Bytes()

		if filterFormat(line) {
			continue // строка уже отформатирована
		}

		// игнорируем форматирование для строк с отступом
		if len(line) > 0 && line[0] == '\t' {
			buf.Write(line)
			buf.WriteByte('\n')

			continue
		}

		// разделяем цветом ключ заголовка и значение
		key, value, _ := bytes.Cut(line, []byte(": "))

		buf.WriteString("[dimgray]") // spellchecker:ignore dimgray
		buf.Write(key)
		buf.WriteByte(':')

		if reNumbers.Match(value) {
			buf.WriteString("[navy]") // цифровое значение
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
		buf.WriteString("[dimgray]Content-Length:[navy]")
		buf.WriteString(strconv.Itoa(len(e.Body)))
		buf.WriteString(clearLine) // сбрасываем форматирование

		buf.WriteByte('\n') // разделитель на тела события
		if !filterFormat([]byte(e.Body)) {
			buf.WriteString(e.Body) // тело события как есть
		}
	}

	return buf.Flush() //nolint:wrapcheck
}

var reNumbers = regexp.MustCompile(`^\s*\d+\.*\d*\s*$`)
