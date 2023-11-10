package main

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
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
		fmt.Printf("Error: %s doesn't exist\n", file)
		return
	}

	if !info.Mode().IsRegular() {
		fmt.Printf("Error: %s is not a file\n", file)
		return
	}

	var src image.Image

	if !copyExif {
		// rotate if necessary
		src, err = imaging.Open(file, imaging.AutoOrientation(true))
		if err != nil {
			fmt.Printf("Error: %v (%s)\n", err, file)
			return
		}
	} else {
		src, err = imaging.Open(file)
		if err != nil {
			fmt.Printf("Error: %v (%s)\n", err, file)
			return
		}
	}

	format, err := imaging.FormatFromFilename(file)

	if err != nil {
		fmt.Printf("Error: cannot detect format: %v\n", err)
		return
	}

	if format.String() == "GIF" {
		// return if GIF is animated - unsupported
		if err := IsGIFAnimated(file); err != nil {
			fmt.Printf("Error: animated GIF not supported (%v)\n", file)
			return
		}
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

	// do not upscale image
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

	tmpFile, err := os.CreateTemp(os.TempDir(), "Goptimized-")

	if err != nil {
		fmt.Printf("Error: cannot create temporary file: %v\n", err)
		return
	}

	defer os.Remove(tmpFile.Name())

	switch imgType := format.String(); imgType {
	case "JPEG":
		err = jpeg.Encode(tmpFile, resized, &jpeg.Options{Quality: quality})
	case "PNG":
		err = png.Encode(tmpFile, resized)
	case "GIF":
		err = gif.Encode(tmpFile, resized, nil)
	case "TIFF":
		err = tiff.Encode(tmpFile, resized, nil)
	case "BMP":
		err = bmp.Encode(tmpFile, resized)
	default:
		fmt.Printf("Error: unsupported file type (%s)\n", file)
		return
	}

	if err != nil {
		fmt.Printf("Error saving output file: %v\n", err)
		return
	}

	// get the temporary filename before closing
	tmpFilename := tmpFile.Name()

	// immediately close the temp file to release pointers
	// so we can modify it with system processes
	tmpFile.Close()

	// run through optimizers
	if format.String() == "JPEG" {
		// run one or the other, running both has no advantage
		if jpegtran != "" {
			RunOptimizer(tmpFilename, true, jpegtran, "-optimize", "-outfile")
		} else if jpegoptim != "" {
			RunOptimizer(tmpFilename, false, jpegoptim, "-f", "-s", "-o")
		}

		if copyExif {
			if err := exifCopy(file, tmpFilename); err != nil {
				fmt.Printf("Error copying exif data: %v (%s)\n", err, file)
				return
			}
		}
	} else if format.String() == "PNG" {
		if pngquant != "" {
			RunOptimizer(tmpFilename, true, pngquant, "-f", "--output")
		}
		if optipng != "" {
			RunOptimizer(tmpFilename, true, optipng, "-out")
		}
	} else if format.String() == "GIF" && gifsicle != "" {
		RunOptimizer(tmpFilename, true, gifsicle, "-o")
	}

	// re-open modified temporary file
	tmpFile, err = os.Open(tmpFilename)
	if err != nil {
		fmt.Printf("Error reopening temporary file: %v\n", err)
		return
	}

	defer tmpFile.Close()

	// get th eoriginal file stats
	srcStat, _ := os.Stat(file)
	srcSize := srcStat.Size()
	// get the optimized file stats
	dstStat, _ := tmpFile.Stat()
	dstSize := dstStat.Size()

	// get the original modification time for later
	mtime := srcStat.ModTime()
	atime := mtime // use mtime as we cannot get atime

	// calculate saved percent
	savedPercent := 100 - math.Round(float64(dstSize)/float64(srcSize)*100)

	if savedPercent > 0 {
		// (over)write the file - not all filesystems support
		// cross-filesystem moving so we overwrite the original
		out, err := os.Create(dstFile)
		if err != nil {
			fmt.Printf("Error opening original file: %v\n", err)
			return
		}

		defer out.Close()

		if _, err := io.Copy(out, tmpFile); err != nil {
			fmt.Printf("Error overwriting original file: %v\n", err)
			return
		}

		if preserveModTimes {
			// transfer original modification times
			if err := os.Chtimes(dstFile, atime, mtime); err != nil {
				fmt.Printf("Error setting file timestamp: %v\n", err)
			}
		}

		fmt.Printf("Goptimized %s (%dx%d %s > %s %v%%)\n", dstFile, resultW, resultH, ByteCountSI(srcSize), ByteCountSI(dstSize), savedPercent)
	} else {
		// if the output directory is not the same,
		// then write a copy of the original file
		if outputDir != "" {
			out, err := os.Create(dstFile)
			if err != nil {
				fmt.Printf("Error opening original file: %v\n", err)
				return
			}

			defer out.Close()

			orig, _ := os.Open(file)

			defer orig.Close()

			if _, err := io.Copy(out, orig); err != nil {
				fmt.Printf("Error ovewriting original file: %v\n", err)
				return
			}

			if preserveModTimes {
				// transfer original modification times
				if err := os.Chtimes(dstFile, atime, mtime); err != nil {
					fmt.Printf("Error setting file timestamp: %v\n", err)
				}
			}

			fmt.Printf("Copied %s (%dx%d %s %v%%)\n", dstFile, srcW, srcH, ByteCountSI(srcSize), 0)
		} else {
			// we didn't actually change anything
			fmt.Printf("Skipped %s (%dx%d %s %v%%)\n", dstFile, srcW, srcH, ByteCountSI(srcSize), 0)
		}
	}

}

// RunOptimizer will run the specified command on a copy of the temporary file,
// and overwrite it if the output is smaller than the original
func RunOptimizer(src string, outFileArg bool, args ...string) {
	// create a new temp file
	tmpFile, err := os.CreateTemp(os.TempDir(), "Goptimized-")

	if err != nil {
		fmt.Printf("Cannot create temporary file: %v\n", err)
		return
	}

	defer os.Remove(tmpFile.Name())

	source, err := os.Open(src)

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
	if outFileArg {
		// most commands require a second filename arg to overwrite the original
		args = append(args, tmpFile.Name())
	}

	// execute the command
	cmd := exec.Command(args[0], args[1:]...)

	if err := cmd.Run(); err != nil {
		fmt.Printf("%s: %v\n", args[0], err)
		return
	}

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

// IsGIFAnimated will return an error if the GIF file has more than 1 frame
func IsGIFAnimated(gifFile string) error {
	file, _ := os.Open(gifFile)
	defer file.Close()

	g, err := gif.DecodeAll(file)
	if err != nil {
		return err
	}

	// Single frame = OK
	if len(g.Image) == 1 {
		return nil
	}

	return fmt.Errorf("Animated gif")
}
