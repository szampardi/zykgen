package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	docopt "github.com/docopt/docopt.go"
	"github.com/szampardi/zykgen"
)

func replaceAtIndex(in string, r rune, i int) string {
	out := []rune(in)
	out[i] = r
	return string(out)
}

const usage = `Zyxel VMG8823-B50B WPA Keygen
 
Usage:
  zykgen (-m|-n|-c) [-b 375000] [-l <length> -L <letter>] <startserial> <endserial>
  zykgen -h | --help
 
Options:
  -b <batch>      Batch size [default 375000].
  -l <length>     Output key length [default: 10].
  -L <letter>     Fifth letter of the serial [default: V].
  -h --help       Show this screen.`

func main() {
	var cocktail zykgen.Cocktail
	var seriale string

	var args struct {
		Sserial string `docopt:"<startserial>"`
		Eserial string `docopt:"<endserial>"`

		Letter       string `docopt:"-L"`
		Length       int    `docopt:"-l"`
		Mojito       bool   `docopt:"-m"`
		Negroni      bool   `docopt:"-n"`
		Cosmopolitan bool   `docopt:"-c"`

		BatchSize int `docopt:"-b"`
	}

	opts, err := docopt.DefaultParser.ParseArgs(usage, os.Args[1:], "")
	if err != nil {
		return
	}

	opts.Bind(&args)
	if args.Mojito {
		cocktail = zykgen.Mojito
	}
	if args.Negroni {
		cocktail = zykgen.Negroni
	}
	if args.Cosmopolitan {
		cocktail = zykgen.Cosmopolitan
	}

	start, err := strconv.Atoi(args.Sserial)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		fmt.Fprintf(os.Stderr, "Serial number should be 12 chars long, (exluding the first char which is 'S'),should contain only numbers, letter 'V' is automatically added")
		return
	}

	end, err := strconv.Atoi(args.Eserial)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		fmt.Fprintf(os.Stderr, "Serial number should be 12 chars long, (exluding the first char which is 'S'),should contain only numbers, letter 'V' is automatically added")
		return
	}

	if start > end {
		fmt.Fprintf(os.Stderr, "End of the serial should be at least > start of the serial")
		return
	}

	if args.BatchSize == 0 {
		args.BatchSize = 375000
	}

	tot := int64(end - start)
	bar := Bar{}
	bar.NewOption(0, tot)
	progress := int64(0)

	if len(args.Sserial) != 12 {
		fmt.Fprintf(os.Stderr, "Serial number should be 12 chars long, (exluding the first char which is 'S'),should contain only numbers, letter 'V' is automatically added ")
		return
	}

	if len(args.Eserial) != 12 {
		fmt.Fprintf(os.Stderr, "Serial number should be 12 chars long, (exluding the first char which is 'S'),should contain only numbers, letter 'V' is automatically added  ")
		return
	}

	wg := &sync.WaitGroup{}
	for i := start; i < end; i++ {
		wg.Add(1)
		progress++
		go func() {
			seriale = fmt.Sprintf("%12d", i)
			seriale = "S" + replaceAtIndex(seriale, []rune(args.Letter)[0], 3)
			fmt.Fprintf(os.Stdout, "%s\n", zykgen.Wpa(seriale, args.Length, cocktail))
			wg.Done()
		}()
		if i%args.BatchSize == 0 {
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
