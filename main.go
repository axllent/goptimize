package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
)

var (
	quality                   int
	maxWidth                  int
	maxHeight                 int
	outputDir                 string
	preserveModificationTimes bool
	jpegoptim                 string
	jpegtran                  string
	optipng                   string
	pngquant                  string
	gifsicle                  string
)

func main() {
	// set the default help
	flag.Usage = func() {
		fmt.Println("Goptimize - downscales and optimizes existing images")
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

	flag.IntVar(&quality, "q", 75, "Quality - JPEG only")
	flag.StringVar(&outputDir, "o", "", "Output directory (default overwrites original)")
	flag.BoolVar(&preserveModificationTimes, "p", true, "Preserve file modification times")
	flag.StringVar(&maxSizes, "m", "", "Downscale to a maximum width & height in pixels (<width>x<height>)")

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

	if len(flag.Args()) < 1 {
		flag.Usage()
		return
	}

	if maxSizes != "" {
		// calculate max sizes from arg[0]
		r := regexp.MustCompile(`^(\d+)(x|X|\*|:)(\d+)$`)
		matches := r.FindStringSubmatch(maxSizes)

		if len(matches) != 4 {
			flag.Usage()
			return
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
				return
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
		// fmt.Printf("  - %s: [undetected]\n", name)
		return err
	}

	fmt.Printf("  - %s: %s\n", name, exe)
	return nil
}
