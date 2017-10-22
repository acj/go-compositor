package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"os"
)

type JPEGReader struct {
	reader io.Reader
	buf   *bytes.Buffer
}

func NewJPEGReader(toRead io.Reader) *JPEGReader {
	return &JPEGReader{io.Reader(toRead), new(bytes.Buffer)}
}

func (r *JPEGReader) ReadImage() (img *image.Image, err error) {
//	jpeg_header := []byte{0xff, 0xd8}
	jpeg_footer := []byte{0xff, 0xd9}
	for {
//		fmt.Printf("Starting with %v bytes\n", r.buf.Len())
		buf_bytes := r.buf.Bytes()
		footer_index := bytes.Index(buf_bytes, jpeg_footer)

		if footer_index == -1 {
			b := make([]byte, 65535)
			n, err := r.reader.Read(b)
			if err != nil && err != io.EOF {
				log.Fatal("Error reading jpeg data: ", err)
			}
//			fmt.Printf("Read %v bytes\n", n)

			if n > 0 {
				r.buf.Write(b[:n])
//				fmt.Printf("Wrote %v bytes\n", n)
			} else {
				// EOF
				break
			}
		} else {
//			fmt.Printf("Found footer at %v\n", footer_index)
			img_bytes := buf_bytes[:(footer_index + 2)]

			// TODO: Verify header and footer

//			fmt.Printf("Decoding %v bytes with start %x and end %x\n", len(img_bytes), img_bytes[0:2], img_bytes[len(img_bytes) - 2:])

			img_buf := bytes.NewBuffer(img_bytes)
			img, err := jpeg.Decode(img_buf)

			if err != nil {
				fmt.Printf("Failed to decode: ", err)

				fo, err := os.Create("failed.jpg")
				if err != nil {
					log.Fatal("Failed to write failure file: ", err)
				}
				defer func() {
					if err := fo.Close(); err != nil {
						log.Fatal("Failed to close failure file: ", err)
					}
				}()

				img_buf.WriteTo(fo)
				return nil, err
			}

			// Set up for the next read
			r.buf = bytes.NewBuffer(buf_bytes[(footer_index + 2):])
//			fmt.Printf("%v bytes for next read\n", r.buf.Len())
//			fmt.Printf("Next bytes are %x\n", buf_bytes[footer_index + 2:footer_index + 4])

			return &img, err
		}
	}

	return nil, io.EOF
}
