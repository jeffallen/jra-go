package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/keep94/sunrise"
)

var lat = flag.Float64("lat", 46.647496, "latitude of the camera")
var lon = flag.Float64("lon", 6.40930383, "longitude of the camera")
var pic = flag.Bool("pic", false, "take a picture now")
var tl = flag.Bool("tl", false, "take a time-lapse now")
var dir = flag.String("dir", ".", "directory to save pictures")

func sunriseTime(now time.Time) time.Time {
	ss := &sunrise.Sunrise{}
	ss.Around(*lat, *lon, now)
	return ss.Sunrise()
}

func sunsetTime(now time.Time) time.Time {
	ss := &sunrise.Sunrise{}
	ss.Around(*lat, *lon, now)
	return ss.Sunset()
}

func picture() {
	now := time.Now()
	fn := fmt.Sprintf("%v/%v.jpg", *dir, now.Format("150405"))
	cmd := exec.Command("raspistill", "-o", fn)
	err := cmd.Run()
	if err != nil {
		log.Println("picture error:", err)
	}
}

func timeLapse(count int, sleep time.Duration) {
	for i := 0; i < count; i++ {
		// Start a timer, so that the time it takes picture() to run
		// is counted in the sleeping time.
		wake := time.After(sleep)
		picture()
		<-wake
	}
}

func main() {
	flag.Parse()

	if *pic {
		picture()
		return
	}

	if *tl {
		timeLapse(10, 10*time.Second)
		return
	}

	log.Printf("Will take %v pictures.", photos)

	for {
		waitForNextEvent()
		timeLapse(photos, period)
	}
}

const before = 10 * time.Minute
const period = 30 * time.Second

// take pictures from sunrise-before until sunrise+before
const photos = int(2 * before / period)

func waitForNextEvent() {
	now := time.Now()
	s := sunriseTime(now).Add(-before)
	fmt.Println("sunrise:", s)
	if now.After(s) {
		// sunrise is past, try sunset
		ss := sunsetTime(now).Add(-before)
		fmt.Println("sunset:", ss)
		if now.After(s) {
			// Sunset is already past today, wait for sunrise tomorrow
			now = now.AddDate(0, 0, 1)
			sr := sunriseTime(now).Add(-before)
			fmt.Println("sunrise:", sr)
			s = sr
		} else {
			s = ss
		}
	}

	sltime := s.Sub(time.Now())
	log.Println("sleep time:", sltime)
	time.Sleep(sltime)
}
