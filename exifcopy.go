// https://stackoverflow.com/a/76779756
package main

import (
	"bufio"
	"errors"
	"io"
	"os"
)

const (
	soi       = 0xD8
	eoi       = 0xD9
	sos       = 0xDA
	exif      = 0xE1
	copyright = 0xEE
	comment   = 0xFE
)

func isMetaTagType(tagType byte) bool {
	// Adapt as needed
	return tagType == exif || tagType == copyright || tagType == comment
}

func copySegments(dst *bufio.Writer, src *bufio.Reader, filterSegment func(tagType byte) bool) error {
	var buf [2]byte
	_, err := io.ReadFull(src, buf[:])
	if err != nil {
		return err
	}
	if buf != [2]byte{0xFF, soi} {
		return errors.New("expected SOI")
	}
	for {
		_, err := io.ReadFull(src, buf[:])
		if err != nil {
			return err
		}
		if buf[0] != 0xFF {
			return errors.New("invalid tag type")
		}
		if buf[1] == eoi {
			// Hacky way to check for EOF
			_, err := src.Read(buf[:1])
			if err != nil && err != io.EOF {
				return err
			}
			// don't return an error as some cameras add the exif data at the end.
			// if n > 0 {
			// 	return errors.New("EOF expected after EOI")
			// }
			return nil
		}
		sos := buf[1] == 0xDA
		filter := filterSegment(buf[1])
		if filter {
			_, err = dst.Write(buf[:])
			if err != nil {
				return err
			}
		}

		_, err = io.ReadFull(src, buf[:])
		if err != nil {
			return err
		}
		if filter {
			_, err = dst.Write(buf[:])
			if err != nil {
				return err
			}
		}

		// Note: Includes the length, but not the tag, so subtract 2
		tagLength := ((uint16(buf[0]) << 8) | uint16(buf[1])) - 2
		if filter {
			_, err = io.CopyN(dst, src, int64(tagLength))
		} else {
			_, err = src.Discard(int(tagLength))
		}
		if err != nil {
			return err
		}
		if sos {
			// Find next tag `FF xx` in the stream where `xx != 0` to skip ECS
			// See https://stackoverflow.com/questions/2467137/parsing-jpeg-file-format-format-of-entropy-coded-segments-ecs
			for {
				bytes, err := src.Peek(2)
				if err != nil {
					return err
				}
				if bytes[0] == 0xFF {
					data, rstMrk := bytes[1] == 0, bytes[1] >= 0xD0 && bytes[1] <= 0xD7
					if !data && !rstMrk {
						break
					}
				}
				if filter {
					err = dst.WriteByte(bytes[0])
					if err != nil {
						return err
					}
				}
				_, err = src.Discard(1)
				if err != nil {
					return err
				}
			}
		}
	}
}

func copyMetadata(outImagePath, imagePath, metadataImagePath string) error {
	outFile, err := os.Create(outImagePath)
	if err != nil {
		return err
	}
	defer outFile.Close()
	writer := bufio.NewWriter(outFile)

	imageFile, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer imageFile.Close()
	imageReader := bufio.NewReader(imageFile)

	metaFile, err := os.Open(metadataImagePath)
	if err != nil {
		return err
	}
	defer metaFile.Close()
	metaReader := bufio.NewReader(metaFile)

	_, err = writer.Write([]byte{0xFF, soi})
	if err != nil {
		return err
	}

	// Copy metadata segments
	// It seems that they need to come first!
	err = copySegments(writer, metaReader, isMetaTagType)
	if err != nil {
		return err
	}
	// Copy all non-metadata segments
	err = copySegments(writer, imageReader, func(tagType byte) bool {
		return !isMetaTagType(tagType)
	})
	if err != nil {
		return err
	}

	_, err = writer.Write([]byte{0xFF, eoi})
	if err != nil {
		return err
	}

	// Flush the writer, otherwise the last couple buffered writes (including the EOI) won't get written!
	return writer.Flush()
}

func exifCopy(fromPath, toPath string) error {
	copyPath := toPath + "~"
	err := os.Rename(toPath, copyPath)
	if err != nil {
		return err
	}
	defer os.Remove(copyPath)
	return copyMetadata(toPath, copyPath, fromPath)
}
