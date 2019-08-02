package main

import (
	"fmt"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/disintegration/imaging"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

// Goptimize downscales and optimizes an existing image
func Goptimize(file string) {
	info, err := os.Stat(file)
	if err != nil {
		fmt.Printf("%s doesn't exist\n", file)
		return
	}
	if !info.Mode().IsRegular() {
		// not a file
		fmt.Printf("%s is not a file\n", file)
		return
	}
	// open original, rotate if neccesary
	src, err := imaging.Open(file, imaging.AutoOrientation(true))
	if err != nil {
		fmt.Printf("%v (%s)\n", err, file)
		return
	}

	format, err := imaging.FormatFromFilename(file)
	if err != nil {
		fmt.Printf("Cannot detect format: %v\n", err)
		return
	}

	outFilename := filepath.Base(file)
	outDir := filepath.Dir(file)
	dstFile := filepath.Join(outDir, outFilename)
	if outputDir != "" {
		dstFile = filepath.Join(outputDir, outFilename)
	}

	// get original image size
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	// Ensure scaling does not upscale image
	imgMaxW := maxWidth
	if imgMaxW == 0 || imgMaxW > srcW {
		imgMaxW = srcW
	}
	imgMaxH := maxHeight
	if imgMaxH == 0 || imgMaxH > srcH {
		imgMaxH = srcH
	}

	resized := imaging.Fit(src, imgMaxW, imgMaxH, imaging.Lanczos)

	dstBounds := resized.Bounds()
	resultW := dstBounds.Dx()
	resultH := dstBounds.Dy()

	tmpFile, err := ioutil.TempFile(os.TempDir(), "Goptimized-")
	if err != nil {
		fmt.Printf("Cannot create temporary file: %v\n", err)
		return
	}
	defer os.Remove(tmpFile.Name())

	if format.String() == "JPEG" {
		err = jpeg.Encode(tmpFile, resized, &jpeg.Options{Quality: quality})
	} else if format.String() == "PNG" {
		err = png.Encode(tmpFile, resized)
	} else if format.String() == "GIF" {
		err = gif.Encode(tmpFile, resized, nil)
	} else if format.String() == "TIFF" {
		err = tiff.Encode(tmpFile, resized, nil)
	} else if format.String() == "BMP" {
		err = bmp.Encode(tmpFile, resized)
	} else {
		fmt.Printf("Cannot Goptimize %s files\n", format.String())
		return
	}
	if err != nil {
		fmt.Printf("Error saving output file: %v\n", err)
		return
	}

	// get the tempoary filename before closing
	tmpFilename := tmpFile.Name()
	// close the temp file to release pointers so we can
	// modify it with system processes
	tmpFile.Close()

	// optimize
	if format.String() == "JPEG" {
		// run one or the other, both don't do anything
		if jpegtran != "" {
			RunOptimiser(tmpFilename, true, jpegtran, "-optimize", "-outfile")
		} else if jpegoptim != "" {
			RunOptimiser(tmpFilename, false, jpegoptim, "-f", "-s", "-o")
		}

	} else if format.String() == "PNG" {
		if pngquant != "" {
			RunOptimiser(tmpFilename, true, pngquant, "-f", "--output")
		}
		if optipng != "" {
			RunOptimiser(tmpFilename, true, optipng, "-out")
		}
	} else if format.String() == "GIF" {
		if gifsicle != "" {
			RunOptimiser(tmpFilename, true, gifsicle, "-o")
		}
	}

	// re-open potentiall modified temporary file
	tmpFile, err = os.Open(tmpFilename)
	if err != nil {
		fmt.Printf("Error reopening temporary file: %v\n", err)
		return
	}

	defer tmpFile.Close()

	// original file stats
	srcStat, _ := os.Stat(file)
	srcSize := srcStat.Size()
	// optimized file stats
	dstStat, _ := tmpFile.Stat()
	dstSize := dstStat.Size()

	// transfer the original file permissions to the new file
	if err = os.Chmod(tmpFile.Name(), srcStat.Mode()); err != nil {
		fmt.Printf("Error setting file permissions: %v\n", err)
		return
	}

	if !skipPreserveModTimes {
		// transfer original modification times
		mtime := srcStat.ModTime()
		atime := mtime // use mtime as we cannot get atime
		if err := os.Chtimes(tmpFile.Name(), atime, mtime); err != nil {
			fmt.Printf("Error setting file timestamp: %v\n", err)
		}
	}

	savedPercent := 100 - math.Round(float64(dstSize)/float64(srcSize)*100)

	if dstSize < srcSize {
		// output is smaller
		if err := os.Rename(tmpFile.Name(), dstFile); err != nil {
			fmt.Printf("Error renaming file: %v\n", err)
			return
		}
		fmt.Printf("Goptimized %s (%dx%d %s/%s %v%%)\n", dstFile, resultW, resultH, ByteCountSI(dstSize), ByteCountSI(srcSize), savedPercent)
	} else {
		if outputDir != "" {
			// just copy the original
			if err := os.Rename(file, dstFile); err != nil {
				fmt.Printf("Error renaming file: %v\n", err)
				return
			}
		}
		// we didn't actually any scaling optimizing
		fmt.Printf("Goptimized %s (%dx%d %s/%s %v%%)\n", dstFile, srcW, srcH, ByteCountSI(srcSize), ByteCountSI(srcSize), 0)
	}

}

// RunOptimiser will run the specified command on a copy of the original file
// and overwrite if the output is smaller than the original
func RunOptimiser(src string, outfile bool, args ...string) {
	// create a new temp file
	tmpFile, err := ioutil.TempFile(os.TempDir(), "Goptimized-")
	if err != nil {
		fmt.Printf("Cannot create temporary file: %v\n", err)
		return
	}
	defer os.Remove(tmpFile.Name())

	source, err := os.Open(src)
	// s, _ := source.Stat()
	// log.Printf("%v\n", s.Size())
	if err != nil {
		fmt.Printf("Cannot open temporary file: %v\n", err)
		return
	}
	defer source.Close()

	if _, err := io.Copy(tmpFile, source); err != nil {
		fmt.Printf("Cannot copy source file: %v\n", err)
		return
	}

	// add the filename to the args
	args = append(args, tmpFile.Name())
	if outfile {
		// most commands require a second filename to overwrite the original
		args = append(args, tmpFile.Name())
	}

	// fmt.Println(args)

	// execute the command
	cmd := exec.Command(args[0], args[1:]...)
	err = cmd.Run()
	// out, err := cmd.Output()
	if err != nil {
		// there was an error
		fmt.Printf("%s: %v\n", args[0], err)
		return
	}
	// fmt.Println(string(out))

	tmpFilename := tmpFile.Name()

	srcStat, _ := source.Stat()
	srcSize := srcStat.Size()
	dstStat, _ := os.Stat(tmpFilename)
	dstSize := dstStat.Size()

	// ensure file pointers are closed before renaming
	tmpFile.Close()
	source.Close()

	if dstSize < srcSize {
		if err := os.Rename(tmpFilename, src); err != nil {
			fmt.Printf("Error renaming file: %v\n", err)
			return
		}
		// fmt.Println(args[0], "=", srcSize, "to", dstSize, "(wrote to", source.Name(), ")")
	} else {
		// fmt.Println(args[0], "!=", srcSize, "to", dstSize)
	}
}

// ByteCountSI returns a human readable size from int64 bytes
func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "kMGTPE"[exp])
}
