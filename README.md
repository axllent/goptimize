# Goptimizer - downscales and optimizes images

Goptimizer is a commandline utility written in Golang. It downscales and optimize existing images JPEG, PNG and Gif files.

Image downscaling is done within Goptimize (`-m <width>x<height>`, see [Usage](#usage-options)), however optimization is done using the following additional tools (if they are installed):

- jpegoptim
- jpegtran (`libjpeg-turbo-progs`)
- optipng
- pngquant
- gifsicle


## Notes

Both `jpegoptim` & `jpegtran` have almost identical optimization, so if both are installed then just `jpegtran` is used for JPG optimization. PNG optimization however will run through both `optipng` & `pngquant` (if installed) as this has definite advantages.

It is highly recommended to install the necessary optimization tools, however they are not required to run goptimize.

Goptimize will remove all exif data from JPEG files, auto-rotating those that relied on it.

It will also preserve (by default) the file's original modification times (`-p=false` to disable).


## Usage options

```
Usage: ./goptimize [options] <images>

Options:
  -gifsicle string
        gifsicle binary (default "gifsicle")
  -jpegoptim string
        jpegoptim binary (default "jpegoptim")
  -jpegtran string
        jpegtran binary (default "jpegtran")
  -m string
        downscale to a maximum width & height in pixels (<width>x<height>)
  -o string
        output directory (default overwrites original)
  -optipng string
        optipng binary (default "optipng")
  -p    preserve file modification times (default true)
  -pngquant string
        pngquant binary (default "pngquant")
  -q int
        quality - JPEG only (default 75)
```


## Examples

- `./goptimize image.png` - optimize a PNG file
- `./goptimize -m 800x800 *` - optimize and downscale all image files to a maximum size of 800x800px
- `./goptimize -m 1200x0 image.jpg` - optimize and downscale a JPG file to a maximum size of width of 1200px
- `./goptimize -o out/ image.jpg` - optimize a JPG file and save it to `out/`
