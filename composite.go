package main

import (
	"bufio"
	"fmt"
	"flag"
	"image"
	"image/draw"
	"image/jpeg"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
)

func main() {
	flag.Parse()
	if len(flag.Args()) != 2 {
		fmt.Println("usage: go-composite <primary.mp4> <secondary.mp4>")
		os.Exit(-1)
	}

	primary_path, secondary_path := flag.Args()[0], flag.Args()[1]

	wg := new(sync.WaitGroup)
	pr_primary, pw_primary := io.Pipe()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer pw_primary.Close()

		cmd := exec.Command("ffmpeg", "-i", primary_path, "-f", "image2pipe", "-vcodec", "mjpeg", "-pix_fmt", "yuv420p", "pipe:1")
		cmd.Stdout = bufio.NewWriter(pw_primary)
		err := cmd.Run()
		if err != nil {
			log.Fatal("ffmpeg error: ", err)
		}
	}()

	pr_secondary, pw_secondary := io.Pipe()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer pw_secondary.Close()

		cmd := exec.Command("ffmpeg", "-i", secondary_path, "-f", "image2pipe", "-vcodec", "mjpeg", "-pix_fmt", "yuv420p", "pipe:1")
		cmd.Stdout = bufio.NewWriter(pw_secondary)
		err := cmd.Run()
		if err != nil {
			log.Fatal("ffmpeg error: ", err)
		}
	}()

	c_primary := make(chan *image.Image, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(c_primary)

		jpeg_r := NewJPEGReader(pr_primary)

		for {
			img, err_read := jpeg_r.ReadImage()
			if err_read != nil {
				if err_read == io.EOF {
					break
				} else {
					log.Fatal(err_read)
				}
			}

			c_primary <- img
		}
	}()

	c_secondary := make(chan *image.Image, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(c_secondary)

		jpeg_r := NewJPEGReader(pr_secondary)

		for {
			img, err_read := jpeg_r.ReadImage()
			if err_read != nil {
				if err_read == io.EOF {
					break
				} else {
					log.Fatal(err_read)
				}
			}

			c_secondary <- img
		}
	}()

	pr_encode, pw_encode := io.Pipe()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer pw_encode.Close()

		for {
			img_primary, more_primary := <-c_primary
			img_secondary, more_secondary := <-c_secondary
			if more_primary && more_secondary {
				bounds := (*img_primary).Bounds()
				r := image.Rectangle{Max: bounds.Max}
				im := image.NewRGBA(r)

				draw.Over.Draw(im, r, *img_primary, image.ZP)
				draw.Over.Draw(im, r, *img_secondary, image.Point{X: bounds.Dx() / 2, Y: 0})

				err_encode := jpeg.Encode(pw_encode, im, nil)
				if err_encode != nil {
					log.Fatal("Error encoding image: ", err_encode)
				}
			} else {
				break
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		cmd := exec.Command("ffmpeg", "-y", "-i", "pipe:0", "-vcodec", "libx264", "-pix_fmt", "yuv420p", "output.mp4")
		cmd.Stdin = bufio.NewReader(pr_encode)
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}()

	wg.Wait()
}
