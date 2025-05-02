package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
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
	Letter *string
	Length *int

	Mojito       *bool
	Negroni      *bool
	Cosmopolitan *bool
	cocktail     zykgen.Cocktail

	BatchSize *int64
	Output    *string
	Gzip      *bool
}

func (c *config) serialize() string {
	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(c); err != nil {
		panic(err)
	}
	return b.String()
}

func (c *config) filename(start, end int64) string {
	if end == 0 || end == start {
		return fmt.Sprintf("zykgen_%d_l%d_S%s.txt", c.cocktail, *c.Length, replaceAtIndex(fmt.Sprintf("%12d", start), []rune(*c.Letter)[0], 3))
	}
	return fmt.Sprintf("zykgen_%d_l%d_S%s-S%s.txt", c.cocktail, *c.Length, replaceAtIndex(fmt.Sprintf("%12d", start), []rune(*c.Letter)[0], 3), replaceAtIndex(fmt.Sprintf("%12d", end), []rune(*c.Letter)[0], 3))
}

var (
	cfg     *config
	outfile *os.File = os.Stdout

	start int64
	end   int64
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
	fname := cfg.filename(start, end)
	switch *cfg.Output {
	case "", "-", os.Stdout.Name():
		if *cfg.Gzip {
			gzw, _ := gzip.NewWriterLevel(outfile, gzip.BestCompression)
			gzw.Comment = cfg.serialize()
			gzw.Name = fname
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
		if *cfg.Output == "auto" {
			*cfg.Output = fname
			if *cfg.Gzip {
				*cfg.Output += ".gz"
			}
		}
		outfile, err = os.OpenFile(*cfg.Output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			panic(err)
		}
		if *cfg.Gzip {
			gzw, _ := gzip.NewWriterLevel(outfile, gzip.BestCompression)
			gzw.Comment = cfg.serialize()
			gzw.Name = fname
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
		fmt.Fprintf(writeTo, "%s\n", zykgen.Wpa(serial, *cfg.Length, cfg.cocktail))
		os.Exit(0)
	case tot < int64(*cfg.BatchSize):
		wg := &sync.WaitGroup{}
		mu := &sync.Mutex{}
		for i := start - 1; i < end; {
			i++
			wg.Add(1)
			go func() {
				serial := "S" + replaceAtIndex(fmt.Sprintf("%12d", i), []rune(*cfg.Letter)[0], 3)
				key := zykgen.Wpa(serial, *cfg.Length, cfg.cocktail)
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
				key := zykgen.Wpa(serial, *cfg.Length, cfg.cocktail)
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
