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

func NewEvent(ev map[string]string) Event {
	// имя события (для CUSTOM используется имя подкласса)
	name, ok := ev["Event-Subclass"]
	if !ok {
		name = ev["Event-Name"]
	}

	// порядковый номер события
	//nolint:errcheck // return 0 on error
	sequence, _ := strconv.ParseInt(ev["Event-Sequence"], 10, 64)

	// время создания события
	var ts time.Time
	if i, err := strconv.ParseInt(ev["Event-Date-Timestamp"], 10, 64); err == nil {
		ts = time.UnixMicro(i)
	}

	// по договоренности тело сообщения в JSON передается с таким именем
	body, ok := ev["_body"]
	if ok {
		delete(ev, "_body")
	}

	// сортируем порядок следования заголовков
	keys := make([]string, 0, len(ev))
	for k := range ev {
		// игнорируем часто используемые дополнительные заголовки
		if _, ok := ignoreHeaders[k]; ok || k == "Content-Length" {
			continue
		}

		keys = append(keys, k)
	}
	slices.Sort(keys)

	// формируем строку с заголовком события
	var w strings.Builder
	//nolint:errcheck // writing to buffer
	for _, k := range keys {
		w.WriteString(k)
		w.WriteString(": ")
		// FIXME: удаляем пробелы и новые строки в начале и конце строки
		// значения заголовка — часто встречается для строк с переносами
		val := strings.TrimSpace(ev[k])
		replaceNewLine.WriteString(&w, val)
		w.WriteByte('\n')
	}

	return Event{
		Name:      name,
		Sequence:  sequence,
		Timestamp: ts,
		Header:    w.String(),
		Body:      body,
	}
}

var replaceNewLine = strings.NewReplacer("\n", "\n\t")

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

	const clearLine = "[-:-:-\n"

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

	return buf.Flush()
}

var reNumbers = regexp.MustCompile(`^\s*\d+\.*\d*\s*$`)
