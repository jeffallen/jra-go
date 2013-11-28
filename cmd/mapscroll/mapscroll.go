package main

import (
	"code.google.com/p/x-go-binding/ui"
	"code.google.com/p/x-go-binding/ui/x11"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"os"
	"time"
)

var game *bool = flag.Bool("game", false, "make it playable")

var rocketPos int
var rocketLimit int
var origin image.Point

const tileHeight = 256
const tileWidth = 256

type flame struct {
	active bool
	where  image.Point
}

var flames [10]flame

func imagePuller(urls chan string, imgs chan *image.Image) {
	for url := range urls {
		r, err := http.Get(url)
		if err == nil {
			//log.Print("Fetched ", url)
			ctype, found := r.Header["Content-Type"]
			if found {
				switch {
				default:
					log.Print("For ", url, ", unknown type: ", ctype)
				case ctype[0] == "image/png":
					img, err := png.Decode(r.Body)
					if err == nil {
						imgs <- &img
					} else {
						log.Print("For ", url, ", decode error: ", err)
					}
				case ctype[0] == "image/jpeg" || ctype[0] == "image/jpg":
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
		url := fmt.Sprintf("http://khm0.google.com/kh/v=106&src=app&x=%d&y=%d&z=16&s=G", x, y)
		urls <- url
		x++
	}
}

//2012/04/11 23:31:13 event: {65362}	keydown arrow up
//2012/04/11 23:31:13 event: {-65362}	keyup arrow up
//2012/04/11 23:31:16 event: {65364}	keydown arrow down
//2012/04/11 23:31:16 event: {-65364}	keyup arrow down

const arrowUp = 65362
const arrowDown = 65364

func rocketAdj(x int) {
	rocketPos += x
	if rocketPos < -rocketLimit {
		rocketPos = -rocketLimit
	}
	if rocketPos > rocketLimit {
		rocketPos = rocketLimit
	}
}

func fire() {
	// find a free flame and make it active
	for i := 0; i < 10; i++ {
		if !flames[i].active {
			flames[i].active = true
			flames[i].where.X = 10
			flames[i].where.Y = rocketPos
			break
		}
	}
}

var debounce int

func processEvent(ch <-chan interface{}) {
	for {
		var ev interface{}
		ok := false
		select {
		case ev, ok = <-ch:
			if !ok {
				log.Fatal("X display closed.")
			}
			switch e := ev.(type) {
			default:
				log.Printf("event: %T", ev)
			case ui.MouseEvent:
				y := origin.Y + rocketPos
				if e.Loc.Y < y {
					rocketAdj(-7)
				} else {
					rocketAdj(7)
				}
				if debounce < 0 && e.Buttons&1 != 0 {
					fire()
					debounce = 5
				}
				debounce--
			case ui.KeyEvent:
				//log.Printf("key: %v", e.Key)
				switch e.Key {
				case arrowUp:
					rocketAdj(-7)
				case arrowDown:
					rocketAdj(7)
				case 'q':
					log.Fatal("Exit.")
				case ' ':
					fire()
				}
			}
		default:
			// no events
			return
		}

		switch ev.(type) {
		case ui.ErrEvent:
			log.Fatal("X11 err: ", ev.(ui.ErrEvent).Err)
		}
	}
}

func main() {
	flag.Parse()

	var rocket image.Image
	rfile, err := os.Open("rocket.png")
	if err == nil {
		rocket, err = png.Decode(rfile)
	}
	if err != nil {
		log.Fatal("Cannot open rocket.png:", err)
	}
	rfile.Close()
	rocketHeight := rocket.Bounds().Dy()

	var flame image.Image
	ffile, err := os.Open("flame.png")
	if err == nil {
		flame, err = png.Decode(ffile)
	}
	if err != nil {
		log.Fatal("Cannot open flame.png:", err)
	}
	ffile.Close()

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

	numTiles := screen.Bounds().Dx()/tileWidth + 2
	tileStrip := image.NewRGBA(image.Rect(0, 0, numTiles*tileWidth, tileHeight))

	// pre-load the tile strip
	for i := 0; i < numTiles; i++ {
		iptr := <-imgReady
		img := *iptr
		draw.Draw(tileStrip, image.Rect(i*tileWidth, 0, i*tileWidth+tileWidth, tileHeight), img, image.ZP, draw.Over)
	}

	rocketLimit = tileHeight/2 - rocketHeight/2
	topBlack := (screen.Bounds().Dy() - tileHeight) / 2
	origin = image.Pt(10, topBlack+tileHeight/2-rocketHeight/2)
	for {
		for x := 0; x < tileWidth; x += 7 {
			then := time.Now()
			draw.Draw(screen, image.Rect(0, topBlack, screen.Bounds().Dx(), topBlack+tileHeight), tileStrip, image.Pt(x, 0), draw.Over)
			if *game {
				draw.Draw(screen, image.Rect(10, topBlack+tileHeight/2-rocketHeight/2+rocketPos, screen.Bounds().Dx(), topBlack+tileHeight), rocket, image.ZP, draw.Over)
				for i := 0; i < 10; i++ {
					if flames[i].active {
						flames[i].where.X += 40
						if flames[i].where.X > screen.Bounds().Dx() {
							flames[i].active = false
						}
						draw.Draw(screen, image.Rectangle{flames[i].where.Add(origin).Add(image.Pt(0, rocketHeight/2)), screen.Bounds().Size()}, flame, image.Pt(0, 0), draw.Over)
					}
				}
			}

			now := time.Now()
			frameTime := now.Sub(then)

			// a flush is just a write on a channel, so it takes negligible time
			xdisp.FlushImage()

			toSleep := 0.1*1e9 - frameTime
			//			log.Print("Took ", frameTime, " ns to draw, will sleep ", toSleep, " ns")
			time.Sleep(toSleep)
			processEvent(xdisp.EventChan())
		}

		// shift tiles in tileStrip and put in new last one
		draw.Draw(tileStrip, image.Rect(0, 0, (numTiles-1)*tileWidth, tileHeight), tileStrip, image.Pt(tileWidth, 0), draw.Over)
		iptr := <-imgReady
		img := *iptr
		draw.Draw(tileStrip, image.Rect((numTiles-1)*tileWidth, 0, numTiles*tileWidth, tileHeight), img, image.ZP, draw.Over)
	}
}

// vim:ts=2
