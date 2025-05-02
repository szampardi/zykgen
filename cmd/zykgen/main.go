package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

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
	Letter       *string
	Length       *int
	Mojito       *bool
	Negroni      *bool
	Cosmopolitan *bool
	BatchSize    *int64

	start    int64
	end      int64
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
		BatchSize:    flag.Int64("b", int64(runtime.NumCPU()*25000), "batch size"),
	}
	flag.Parse()
	if *cfg.Mojito {
		cfg.cocktail = zykgen.Mojito
	} else if *cfg.Negroni {
		cfg.cocktail = zykgen.Negroni
	} else {
		cfg.cocktail = zykgen.Cosmopolitan
	}
	var err error
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
		cfg.start, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		cfg.end = cfg.start
	case 2:
		if err := checkLength(args[0], 12); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(127)
		}
		cfg.start, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		if err := checkLength(args[1], 12); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(127)
		}
		cfg.end, err = strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		if cfg.start > cfg.end {
			fmt.Fprintf(os.Stderr, "end serial should be at least > start")
			os.Exit(128)
		}
	default:
		flag.Usage()
		os.Exit(127)
	}
}

func main() {
	now := time.Now()
	tot := int64(cfg.end-cfg.start) + 1
	switch {
	case tot == 1:
		serial := fmt.Sprintf("%12d", cfg.start)
		serial = "S" + replaceAtIndex(serial, []rune(*cfg.Letter)[0], 3)
		fmt.Fprintf(os.Stdout, "%s\n", zykgen.Wpa(serial, *cfg.Length, cfg.cocktail))
		os.Exit(0)
	case tot < int64(*cfg.BatchSize):
		wg := &sync.WaitGroup{}
		for i := cfg.start - 1; i < cfg.end; {
			i++
			wg.Add(1)
			go func() {
				serial := fmt.Sprintf("%12d", i)
				serial = "S" + replaceAtIndex(serial, []rune(*cfg.Letter)[0], 3)
				fmt.Fprintf(os.Stdout, "%s\n", zykgen.Wpa(serial, *cfg.Length, cfg.cocktail))
				wg.Done()
			}()
		}
		wg.Wait()
	default:
		bar := Bar{}
		bar.NewOption(0, tot)
		progress := int64(0)

		wg := &sync.WaitGroup{}
		for i := cfg.start - 1; i < cfg.end; {
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
	fmt.Fprintf(os.Stderr, "computed %d keys in %s\n", tot, time.Since(now).String())
}
