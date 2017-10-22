package main

import (
	"bufio"
	"image/jpeg"
	"io"
	"log"
	"os/exec"
	"sync"
)

func main() {
	wg := new(sync.WaitGroup)

	pr, pw := io.Pipe()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer pw.Close()

		cmd := exec.Command("ffmpeg", "-i", "testdata/short.mp4", "-f", "image2pipe", "-vcodec", "mjpeg", "-pix_fmt", "yuv420p", "pipe:1")
		cmd.Stdout = bufio.NewWriter(pw)
		err := cmd.Run()
		if err != nil {
			log.Fatal("ffmpeg error: ", err)
		}
	}()

	pr2, pw2 := io.Pipe()

	// ffmpeg to JPEG reader
	jpeg_r := NewJPEGReader(pr)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer pw2.Close()

		for {
			img, err_read := jpeg_r.ReadImage()
			if err_read != nil {
				if err_read == io.EOF {
					break
				} else {
					log.Fatal(err_read)
				}
			}

			err_encode := jpeg.Encode(pw2, img, nil)
			if err_encode != nil {
				log.Fatal("Error encoding image: ", err_encode)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		cmd := exec.Command("ffmpeg", "-y", "-i", "pipe:0", "-vcodec", "libx264", "-pix_fmt", "yuv420p", "output.mp4")
		cmd.Stdin = bufio.NewReader(pr2)
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}()

	wg.Wait()
}

