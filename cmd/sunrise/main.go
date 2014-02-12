package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/keep94/sunrise"
)

var lat = flag.Float64("lat", 46.647496, "latitude of the camera")
var lon = flag.Float64("lon", 6.40930383, "longitude of the camera")
var nphotos = flag.Int("nphotos", 0, "how many photos to take (debug only)")
var pic = flag.Bool("pic", false, "take a picture now")
var tl = flag.Bool("tl", false, "take a time-lapse now")
var dir = flag.String("dir", ".", "directory to save pictures")
var beforeTL = flag.String("beforeTL", "before-tl", "script to run before time lapse")
var afterTL = flag.String("afterTL", "after-tl", "script to run after time lapse")
var before = flag.Duration("before", 45*time.Minute, "how long to start taking photos before sunrise/sunset")
var period = flag.Duration("period", 30*time.Second, "how long to wait between photos")

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

func run(exe string) {
	cmd := exec.Command(exe)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Println("run error:", err)
	}
}

func picture() {
	now := time.Now()
	fn := fmt.Sprintf("%v/%v.jpg", *dir, now.Format("150405"))
	log.Println("photo:", fn)
	cmd := exec.Command("raspistill", "-n", "--width", "1920",
		"--height", "1080", "-ex", "night", "-awb", "horizon", "-o", fn)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Println("picture error:", err)
	}
}

func timeLapse(count int, sleep time.Duration) {
	run(*beforeTL)
	for i := 0; i < count; i++ {
		// Start a timer, so that the time it takes picture() to run
		// is counted in the sleeping time.
		wake := time.After(sleep)
		picture()
		<-wake
	}
	run(*afterTL)
}

func main() {
	flag.Parse()

	photos := *nphotos
	if photos == 0 {
		// auto: take pictures from sunrise-before until sunrise+before
		photos = int(2 * *before / *period)
	}

	// "unit tests": -pic (one picture) or -tl (time lapse)
	if *pic {
		picture()
		return
	}
	if *tl {
		timeLapse(photos, *period)
		return
	}

	log.Printf("Will take %v pictures, one each %v.", photos, *period)

	for {
		waitForNextEvent()
		timeLapse(photos, *period)
	}
}

func waitForNextEvent() {
	l,  _ := time.LoadLocation("Local")
	now := time.Now()

	// A sunrise one?
	s := sunriseTime(now).Add(-*before)
	if now.Before(s) {
		goto ok
	}

	// Take a noon time lapse
	s = time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, l)
	s = s.Add(-*before)
	if now.Before(s) {
		goto ok
	}

	// Sunrise is past, try sunset.
	s = sunsetTime(now).Add(-*before)
	if now.Before(s) {
		goto ok
	}

	// Sunset is already past today, wait for sunrise tomorrow
	now = now.AddDate(0, 0, 1)
	s = sunriseTime(now).Add(-*before)

ok:
	sltime := s.Sub(time.Now())
	log.Println("next event is at", s, ", sleeping", sltime)
	time.Sleep(sltime)
}
