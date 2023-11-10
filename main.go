package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"sync"

	"github.com/axllent/ghru"
	"github.com/spf13/pflag"
)

var (
	quality          int
	maxWidth         int
	maxHeight        int
	outputDir        string
	preserveModTimes bool
	jpegoptim        string
	jpegtran         string
	optipng          string
	pngquant         string
	gifsicle         string
	copyExif         bool
	threads          = 1
	version          = "dev"
)

func main() {
	// set up new flag instance
	flag := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	// set the default help
	flag.Usage = func() {
		fmt.Println("Goptimize - downscales and optimizes images")
		fmt.Printf("\nUsage: %s [options] <images>\n", os.Args[0])
		fmt.Println("\nOptions:")
		flag.SortFlags = false
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Printf("  %s image.png\n", os.Args[0])
		fmt.Printf("  %s -m 800x800 *.jpg\n", os.Args[0])
		fmt.Printf("  %s -o out/ -q 90 -m 1600x1600 *.jpg\n", os.Args[0])

		fmt.Println("\nDetected optimizers:")
		if err := displayDetectedOptimizer("jpegtran ", jpegtran); err != nil {
			displayDetectedOptimizer("jpegoptim", jpegoptim)
		}
		displayDetectedOptimizer("optipng  ", optipng)
		displayDetectedOptimizer("pngquant ", pngquant)
		displayDetectedOptimizer("gifsicle ", gifsicle)
	}

	var maxSizes string
	var multiThreaded, update, showversion, showhelp bool

	flag.IntVarP(&quality, "quality", "q", 75, "quality, JPEG only")
	flag.StringVarP(&maxSizes, "max", "m", "", "downscale to a maximum width & height in pixels (<width>x<height>)")
	flag.StringVarP(&outputDir, "out", "o", "", "output directory (default overwrites original)")
	flag.BoolVarP(&preserveModTimes, "preserve", "p", true, "preserve file modification times")
	flag.BoolVarP(&copyExif, "exif", "e", false, "copy exif data")
	flag.BoolVarP(&update, "update", "u", false, "update to latest release")
	flag.BoolVarP(&multiThreaded, "threaded", "t", false, "run multi-threaded (use all CPU cores)")
	flag.BoolVarP(&showversion, "version", "v", false, "show version number")
	flag.BoolVarP(&showhelp, "help", "h", false, "show help")

	// third-party optimizers
	flag.StringVar(&jpegtran, "jpegtran", "jpegtran", "jpegtran binary")
	flag.StringVar(&jpegoptim, "jpegoptim", "jpegoptim", "jpegoptim binary")
	flag.StringVar(&gifsicle, "gifsicle", "gifsicle", "gifsicle binary")
	flag.StringVar(&pngquant, "pngquant", "pngquant", "pngquant binary")
	flag.StringVar(&optipng, "optipng", "optipng", "optipng binary")

	flag.SortFlags = false

	// parse args excluding os.Args[0]
	flag.Parse(os.Args[1:])

	// detect optimizer paths
	gifsicle, _ = exec.LookPath(gifsicle)
	jpegoptim, _ = exec.LookPath(jpegoptim)
	jpegtran, _ = exec.LookPath(jpegtran)
	optipng, _ = exec.LookPath(optipng)
	pngquant, _ = exec.LookPath(pngquant)

	if showhelp {
		flag.Usage()
		os.Exit(1)
	}

	if showversion {
		fmt.Println(fmt.Sprintf("Version: %s", version))
		latest, _, _, err := ghru.Latest("axllent/goptimize", "goptimize")
		if err == nil && ghru.GreaterThan(latest, version) {
			fmt.Printf("Update available: %s\nRun `%s -u` to update.\n", latest, os.Args[0])
		}
		return
	}

	if update {
		rel, err := ghru.Update("axllent/goptimize", "goptimize", version)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("Updated %s to version %s\n", os.Args[0], rel)
		return
	}

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	if maxSizes != "" {
		// calculate max sizes from arg[0]
		r := regexp.MustCompile(`^(\d+)(x|X|\*|:)(\d+)$`)
		matches := r.FindStringSubmatch(maxSizes)

		if len(matches) != 4 {
			flag.Usage()
			os.Exit(1)
		}

		maxWidth, _ = strconv.Atoi(matches[1])
		maxHeight, _ = strconv.Atoi(matches[3])
	}

	// parse arguments
	args := flag.Args()

	if outputDir != "" {
		// ensure the output directory exists
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			err := os.MkdirAll(outputDir, os.ModePerm)
			if err != nil {
				fmt.Printf("Cannot create output directory: %s\n", outputDir)
				os.Exit(1)
			}
		}
	}

	if multiThreaded {
		threads = runtime.NumCPU()
	}

	processChan := make(chan string)

	wg := &sync.WaitGroup{} // Signal to main goroutine that worker goroutines are working/done working
	wg.Add(threads)

	for i := 0; i < threads; i++ {
		go func() {
			for nextFile := range processChan {
				Goptimize(nextFile)
			}
			// Channel was closed, so we finished this goroutine.
			wg.Done() // Let main goroutine know we are done.
		}()
	}

	for _, img := range args {
		processChan <- img
	}

	// Close the channel.  This tells the worker goroutines that they can be done.
	close(processChan)

	// Wait for all worker goroutines to finish processing the IPs
	wg.Wait()
}

// displayDetectedOptimizer prints whether the optimizer was found
func displayDetectedOptimizer(name, bin string) error {
	exe, err := exec.LookPath(bin)
	if err != nil {
		return err
	}

	fmt.Printf("  - %s: %s\n", name, exe)
	return nil
}
