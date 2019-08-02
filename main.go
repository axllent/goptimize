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
	quality              int
	maxWidth             int
	maxHeight            int
	outputDir            string
	skipPreserveModTimes bool
	jpegoptim            string
	jpegtran             string
	optipng              string
	pngquant             string
	gifsicle             string
)

func main() {
	// modify the default help
	flag.Usage = func() {
		fmt.Println("Re-save & resample images, with optional optimization.")
		fmt.Printf("\nUsage: %s [options] <images>\n", os.Args[0])
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Printf("  %s -s 800x800 *.jpg\n", os.Args[0])
		fmt.Printf("  %s -o out/ -q 90 -s 1600x1600 *.jpg\n", os.Args[0])

		fmt.Println("\nOtimizers:")
		if err := displayDelectedOptimizer("jpegtran ", jpegtran); err != nil {
			displayDelectedOptimizer("jpegoptim", jpegoptim)
		}
		displayDelectedOptimizer("optipng  ", optipng)
		displayDelectedOptimizer("pngquant ", pngquant)
		displayDelectedOptimizer("gifsicle ", gifsicle)
	}

	var maxSizes string

	flag.IntVar(&quality, "q", 75, "Quality - affects jpeg only")
	flag.StringVar(&outputDir, "o", "", "Output directory (default overwrites original)")
	flag.BoolVar(&skipPreserveModTimes, "n", false, "Do not preserve file modification times")
	flag.StringVar(&maxSizes, "m", "", "Scale down to a maximum width & height. Format must be <width>x<height>.")

	// third-party optimizers
	flag.StringVar(&gifsicle, "gifsicle", "gifsicle", "Alternative gifsicle name")
	flag.StringVar(&jpegoptim, "jpegoptim", "jpegoptim", "Alternative jpegoptim name")
	flag.StringVar(&jpegtran, "jpegtran", "jpegtran", "Alternative jpegtran name")
	flag.StringVar(&optipng, "optipng", "optipng", "Alternative optipng name")
	flag.StringVar(&pngquant, "pngquant", "pngquant", "Alternative pngquant name")

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
