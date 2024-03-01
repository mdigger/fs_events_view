package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	// разбираем параметры командной строки
	filename := flag.String("file", "events.log", "path to events log file")
	allHeaders := flag.Bool("all-headers", false, "show all event headers")
	hideEventsStr := flag.String("hide-events", "", "comma-separated list of event names to hide")
	search := flag.String("search", "", "search regexp string")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"Usage\n  %s: [options] [search-filter]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}
	flag.Parse()

	// если нужно показать все заголовки
	if *allHeaders {
		ignoreHeaders = nil
	}

	// разбираем лог событий
	events, err := Parse(*filename)
	if err != nil {
		return err
	}

	// разбираем список событий для фильтрации
	hideEvents := make([]string, 0, strings.Count(*hideEventsStr, ","))
	for _, name := range strings.Split(*hideEventsStr, ",") {
		hideEvents = append(hideEvents, strings.TrimSpace(name))
	}

	// запускаем приложение
	return NewApp(events, *search, hideEvents...).Run()
}
