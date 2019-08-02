package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/axllent/gitrel"
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
	version          = "dev"
)

func main() {
	// set the default help
	flag.Usage = func() {
		fmt.Println("Goptimize - downscales and optimizes images")
		fmt.Printf("\nUsage: %s [options] <images>\n", os.Args[0])
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Printf("  %s image.png\n", os.Args[0])
		fmt.Printf("  %s -m 800x800 *.jpg\n", os.Args[0])
		fmt.Printf("  %s -o out/ -q 90 -m 1600x1600 *.jpg\n", os.Args[0])

		fmt.Println("\nDetected optimizers:")
		if err := displayDelectedOptimizer("jpegtran ", jpegtran); err != nil {
			displayDelectedOptimizer("jpegoptim", jpegoptim)
		}
		displayDelectedOptimizer("optipng  ", optipng)
		displayDelectedOptimizer("pngquant ", pngquant)
		displayDelectedOptimizer("gifsicle ", gifsicle)
	}

	var maxSizes string
	var update, showversion bool

	flag.IntVar(&quality, "q", 75, "quality - JPEG only")
	flag.StringVar(&outputDir, "o", "", "output directory (default overwrites original)")
	flag.BoolVar(&preserveModTimes, "p", true, "preserve file modification times")
	flag.StringVar(&maxSizes, "m", "", "downscale to a maximum width & height in pixels (<width>x<height>)")
	flag.BoolVar(&update, "u", false, "update to latest release")
	flag.BoolVar(&showversion, "v", false, "show version number")

	// third-party optimizers
	flag.StringVar(&gifsicle, "gifsicle", "gifsicle", "gifsicle binary")
	flag.StringVar(&jpegoptim, "jpegoptim", "jpegoptim", "jpegoptim binary")
	flag.StringVar(&jpegtran, "jpegtran", "jpegtran", "jpegtran binary")
	flag.StringVar(&optipng, "optipng", "optipng", "optipng binary")
	flag.StringVar(&pngquant, "pngquant", "pngquant", "pngquant binary")

	// parse flags
	flag.Parse()

	// detect optimizer paths
	gifsicle, _ = exec.LookPath(gifsicle)
	jpegoptim, _ = exec.LookPath(jpegoptim)
	jpegtran, _ = exec.LookPath(jpegtran)
	optipng, _ = exec.LookPath(optipng)
	pngquant, _ = exec.LookPath(pngquant)

	if showversion {
		fmt.Println(fmt.Sprintf("Version: %s", version))
		latest, _, _, err := gitrel.Latest("axllent/goptimize", "goptimize")
		if err == nil && latest != version {
			fmt.Printf("Update available: %s\nRun `%s -u` to update.\n", latest, os.Args[0])
		}
		return
	}

	if update {
		rel, err := gitrel.Update("axllent/goptimize", "goptimize", version)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("Updated %s to version %s", os.Args[0], rel)
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

	for _, img := range args {
		Goptimize(img)
	}
}

// displayDelectedOptimizer prints whether the optimizer was found
func displayDelectedOptimizer(name, bin string) error {
	exe, err := exec.LookPath(bin)
	if err != nil {
		return err
	}

	fmt.Printf("  - %s: %s\n", name, exe)
	return nil
}
