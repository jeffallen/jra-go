// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"fmt"
	"time"
	"http"
	"exp/draw"
	"exp/draw/x11"
	"image"
	"image/png"
	"image/jpeg"
)

func imagePuller(urls chan string, imgs chan *image.Image) {
	for url := range urls {
		r, _, err := http.Get(url)
		if err == nil {
			log.Print("Fetched ", url)
			ctype, found := r.Header["Content-Type"]
			if found {
				switch {
				default:
					log.Print("For ", url, ", unknown type: ", ctype)
				case ctype == "image/png":
					img, err := png.Decode(r.Body)
					if err == nil {
						imgs <- &img
					} else {
						log.Print("For ", url, ", decode error: ", err)
					}
				case ctype == "image/jpeg" || ctype == "image/jpg":
					img, err := jpeg.Decode(r.Body)
					if err == nil {
						imgs <- &img
					} else {
						log.Print("For ", url, ", decode error: ", err)
						return
					}
				}
			} else {
				log.Print("For ", url, ", no content type.")
			}
			r.Body.Close()
		} else {
			log.Print("Error fetching ", url, ": ", err)
		}
	}
}

func urlGen(urls chan string) {
	x := 33981
	y := 23179
	for {
		url := fmt.Sprintf("http://khm1.google.com/kh/v=74&x=%d&s=&y=%d&z=16&s=Ga", x, y)
		urls <- url
		x++
	}
}

func processEvent(ch <-chan interface {}) {
	for {
		if closed(ch) {
			log.Fatal("X display closed.")
		}
		var ev interface {}
		select {
			case ev = <-ch:
				// ok, continue
			default:
				// no events
				return
		}

		switch ev.(type) {
		case draw.ErrEvent:
			log.Fatal("X11 err: ", ev.(draw.ErrEvent).Err)
		}
	}
}

func main () {
	urls := make(chan string, 4)
	imgReady := make(chan *image.Image, 4)

	go imagePuller(urls, imgReady)
	go urlGen(urls)

	xdisp, err := x11.NewWindow()
	if err != nil {
		log.Fatal("X11 error: ", err)
	}
	screen := xdisp.Screen()

//2010/11/29 16:23:17 one tile is sized (0,0)-(256,256)
//2010/11/29 16:23:17 the screen is sized (0,0)-(800,600)

//	log.Print("one tile is sized ", img.Bounds())
//	log.Print("the screen is sized ", screen.Bounds())

	tileHeight := 256
	tileWidth := 256
	numTiles := screen.Bounds().Dx() / tileWidth + 2
	tileStrip := image.NewRGBA(numTiles * tileWidth, tileHeight)

	// pre-load the tile strip
	for i := 0; i < numTiles; i++ {
		iptr := <- imgReady
		img := *iptr
		draw.Draw(tileStrip, image.Rect(i * tileWidth, 0, i * tileWidth+tileWidth, tileHeight), img, image.ZP)
	}
	
	topBlack := (screen.Bounds().Dy() - tileHeight)/ 2
	for {
		for x := 0; x < tileWidth; x += 15 {
			then := time.Nanoseconds()
			draw.Draw(screen, image.Rect(0, topBlack, screen.Bounds().Dx(), topBlack+tileHeight), tileStrip, image.Pt(x, 0))
			now := time.Nanoseconds()
			frameTime := now - then

			// a flush is just a write on a channel, so it takes negligible time
			xdisp.FlushImage()

			toSleep := 0.1 * 1e9 - frameTime
//			log.Print("Took ", frameTime, " ns to draw, will sleep ", toSleep, " ns")
			time.Sleep(toSleep)
			processEvent(xdisp.EventChan())
		}

		// shift tiles in tileStrip and put in new last one
		draw.Draw(tileStrip, image.Rect(0, 0, (numTiles-1)*tileWidth, tileHeight), tileStrip, image.Pt(tileWidth, 0))
		iptr := <- imgReady
		img := *iptr
		draw.Draw(tileStrip, image.Rect((numTiles-1)*tileWidth, 0, numTiles*tileWidth, tileHeight), img, image.ZP)
	}
}
