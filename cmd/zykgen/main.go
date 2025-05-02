package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/szampardi/zykgen"
)

func replaceAtIndex(in string, r rune, i int) string {
	out := []rune(in)
	out[i] = r
	return string(out)
}

func checkLength(in string, l int) error {
	if len(in) != l {
		return fmt.Errorf("invalid serial number [%s], should be 12 chars long (exluding the first char which is 'S'), should contain only numbers, given letter is automatically inserted at 5th position", in)
	}
	return nil
}

type config struct {
	Letter       *string `docopt:"-L"`
	Length       *int    `docopt:"-l"`
	Mojito       *bool   `docopt:"-m"`
	Negroni      *bool   `docopt:"-n"`
	Cosmopolitan *bool   `docopt:"-c"`
	BatchSize    *int    `docopt:"-b"`

	cocktail zykgen.Cocktail
}

var cfg *config

func init() {
	cfg = &config{
		Letter:       flag.String("L", "V", "fifth letter of the serial"),
		Length:       flag.Int("l", 10, "output key length"),
		Mojito:       flag.Bool("m", false, "algorithm"),
		Negroni:      flag.Bool("n", false, "algorithm"),
		Cosmopolitan: flag.Bool("c", true, "algorithm"),
		BatchSize:    flag.Int("b", 375000, "batch size"),
	}
	flag.Parse()
	if *cfg.Mojito {
		cfg.cocktail = zykgen.Mojito
	} else if *cfg.Negroni {
		cfg.cocktail = zykgen.Negroni
	} else {
		cfg.cocktail = zykgen.Cosmopolitan
	}
}

func main() {
	var err error
	var start, end int
	args := flag.Args()
	switch len(args) {
	case 0:
		flag.Usage()
		os.Exit(127)
	case 1:
		if err = checkLength(args[0], 12); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(127)
		}
		start, err = strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		end = start
	case 2:
		if err := checkLength(args[0], 12); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(127)
		}
		start, err = strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		if err := checkLength(args[1], 12); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(127)
		}
		end, err = strconv.Atoi(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		if start > end {
			fmt.Fprintf(os.Stderr, "end serial should be at least > start")
			os.Exit(128)
		}
	default:
		flag.Usage()
		os.Exit(127)
	}

	tot := int64(end-start) + 1
	if tot == 0 {
		serial := fmt.Sprintf("%12d", start)
		serial = "S" + replaceAtIndex(serial, []rune(*cfg.Letter)[0], 3)
		fmt.Fprintf(os.Stdout, "%s\n", zykgen.Wpa(serial, *cfg.Length, cfg.cocktail))
		os.Exit(0)
	}

	bar := Bar{}
	bar.NewOption(0, tot)
	progress := int64(0)

	wg := &sync.WaitGroup{}
	for i := start - 1; i < end; {
		i++
		wg.Add(1)
		progress++
		go func() {
			serial := fmt.Sprintf("%12d", i)
			serial = "S" + replaceAtIndex(serial, []rune(*cfg.Letter)[0], 3)
			fmt.Fprintf(os.Stdout, "%s\n", zykgen.Wpa(serial, *cfg.Length, cfg.cocktail))
			wg.Done()
		}()
		if i%*cfg.BatchSize == 0 {
			wg.Wait()
			bar.Play(progress)
		}
	}
	wg.Wait()
	bar.Play(progress)
	bar.Finish()
}

// Bar ...
type Bar struct {
	percent int64  // progress percentage
	cur     int64  // current progress
	total   int64  // total value for progress
	rate    string // the actual progress bar to be printed
	graph   string // the fill value for progress bar
}

func (bar *Bar) NewOption(start, total int64) {
	bar.cur = start
	bar.total = total
	if bar.graph == "" {
		bar.graph = ">"
	}
	bar.percent = bar.getPercent()
	for i := 0; i < int(bar.percent); i += 2 {
		bar.rate += bar.graph // initial progress position
	}
}

func (bar *Bar) getPercent() int64 {
	return int64((float32(bar.cur) / float32(bar.total)) * 100)
}

func (bar *Bar) Play(cur int64) {
	bar.cur = cur
	last := bar.percent
	bar.percent = bar.getPercent()
	if bar.percent != last && bar.percent%2 == 0 {
		bar.rate += bar.graph
	}
	fmt.Fprintf(os.Stderr, "\r[%-50s]%3d%% %8d/%d", bar.rate, bar.percent, bar.cur, bar.total)
}

func (bar *Bar) Finish() {
	fmt.Fprintln(os.Stderr)
}
