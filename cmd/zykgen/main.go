package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io"
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

	BatchSize *int64
	Output    *string
	Gzip      *bool
}

var (
	cfg      *config
	outfile  *os.File = os.Stdout
	cocktail zykgen.Cocktail
	start    int64
	end      int64
)

func init() {
	cfg = &config{
		Letter:       flag.String("L", "V", "fifth letter of the serial"),
		Length:       flag.Int("l", 10, "output key length"),
		Mojito:       flag.Bool("m", false, "algorithm"),
		Negroni:      flag.Bool("n", false, "algorithm"),
		Cosmopolitan: flag.Bool("c", true, "algorithm"),

		BatchSize: flag.Int64("b", int64(runtime.NumCPU()*10000), "batch size"),
		Output:    flag.String("o", "-", "write to"),
		Gzip:      flag.Bool("g", false, "write gzipped"),
	}
	flag.Parse()
	if *cfg.Mojito {
		cocktail = zykgen.Mojito
	} else if *cfg.Negroni {
		cocktail = zykgen.Negroni
	} else {
		cocktail = zykgen.Cosmopolitan
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
		start, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		end = start
	case 2:
		if err := checkLength(args[0], 12); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(127)
		}
		start, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		if err := checkLength(args[1], 12); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(127)
		}
		end, err = strconv.ParseInt(args[1], 10, 64)
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
}

func main() {
	var err error
	var writeTo io.Writer
	switch *cfg.Output {
	case "", "-", os.Stdout.Name():
		if *cfg.Gzip {
			gzw, _ := gzip.NewWriterLevel(outfile, gzip.BestCompression)
			defer func() {
				if err = gzw.Flush(); err != nil {
					panic(err)
				}
				if err = gzw.Close(); err != nil {
					panic(err)
				}
			}()
			writeTo = gzw
		} else {
			writeTo = os.Stdout
		}
	default:
		outfile, err = os.OpenFile(*cfg.Output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			panic(err)
		}
		if *cfg.Gzip {
			gzw, _ := gzip.NewWriterLevel(outfile, gzip.BestCompression)
			defer func() {
				if err = gzw.Flush(); err != nil {
					panic(err)
				}
				if err = gzw.Close(); err != nil {
					panic(err)
				}
				if err = outfile.Close(); err != nil {
					panic(err)
				}
			}()
			writeTo = gzw
		} else {
			defer func() {
				if err = outfile.Close(); err != nil {
					panic(err)
				}
			}()
			writeTo = outfile
		}
	}
	now := time.Now()
	tot := int64(end-start) + 1
	switch {
	case tot == 1:
		serial := fmt.Sprintf("%12d", start)
		serial = "S" + replaceAtIndex(serial, []rune(*cfg.Letter)[0], 3)
		fmt.Fprintf(writeTo, "%s\n", zykgen.Wpa(serial, *cfg.Length, cocktail))
		os.Exit(0)
	case tot < int64(*cfg.BatchSize):
		wg := &sync.WaitGroup{}
		mu := &sync.Mutex{}
		for i := start - 1; i < end; {
			i++
			wg.Add(1)
			go func() {
				serial := "S" + replaceAtIndex(fmt.Sprintf("%12d", i), []rune(*cfg.Letter)[0], 3)
				key := zykgen.Wpa(serial, *cfg.Length, cocktail)
				mu.Lock()
				fmt.Fprintf(writeTo, "%s\n", key)
				mu.Unlock()
				wg.Done()
			}()
		}
		wg.Wait()
	default:
		bar := Bar{}
		bar.NewOption(0, tot)
		progress := int64(0)
		wg := &sync.WaitGroup{}
		mu := &sync.Mutex{}
		for i := start - 1; i < end; {
			i++
			wg.Add(1)
			progress++
			go func() {
				serial := "S" + replaceAtIndex(fmt.Sprintf("%12d", i), []rune(*cfg.Letter)[0], 3)
				key := zykgen.Wpa(serial, *cfg.Length, cocktail)
				mu.Lock()
				fmt.Fprintf(writeTo, "%s\n", key)
				mu.Unlock()
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
