package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
)

func Parse(filename string) ([]Event, error) {
	// открываем файл с логами
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// подсчитываем количество записей в файле и сбрасываем позицию чтения на начало
	var lineCount int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineCount++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	return decode(file, lineCount)
}

// словарь для подсказок названий ключей в заголовке событий.
var headers map[string]struct{}

func decode(r io.Reader, length int) ([]Event, error) {
	// выделяем память под записи с событиями
	events := make([]Event, 0, length)
	headers = make(map[string]struct{})
	// разбираем события из файле
	dec := json.NewDecoder(r)
	for {
		// декодируем описание события
		ev := make(map[string]string)
		if err := dec.Decode(&ev); err != nil {
			if errors.Is(err, io.EOF) {
				break // больше событий нет
			}

			return nil, err
		}

		// добавляем событие в общий список
		events = append(events, NewEvent(ev))

		// запоминаем названия заголовков событий
		for k := range ev {
			if _, ok := ignoreHeaders[k]; ok || k == "Content-Length" {
				continue
			}
			if _, ok := headers[k]; !ok {
				headers[k] = struct{}{}
			}
		}
	}

	return events, nil
}
