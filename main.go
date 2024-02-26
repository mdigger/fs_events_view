package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
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
	allHeaders := flag.Bool("all", false, "show all event headers")
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

	// формируем список названий заголовков для подстановки
	autocomplete := make([]string, 0, len(headers))
	for k := range headers {
		autocomplete = append(autocomplete, k+": ")
	}

	slices.Sort(autocomplete)

	// запускаем приложение
	return NewApp(events, autocomplete, flag.Args()...).Run()
}
