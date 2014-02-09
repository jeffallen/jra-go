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

func tomorrowSunrise() time.Time {
	now := time.Now()
	if now.Hour() < 6 {
		// ooh, someone's up late; if we are being started between 0 and 6am
		// don't add to tomorrow
	} else {
		now = now.AddDate(0, 0, 1)
	}

	ss := &sunrise.Sunrise{}
	ss.Around(*lat, *lon, now)
	return ss.Sunrise()
}

func todaySunset() time.Time {
	now := time.Now()
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
		timeLapse(10, 10 * time.Second)
		return
	}

	before := 10 * time.Minute
	period := 30 * time.Second
	// take pictures from sunrise-before until sunrise+before
	photos := int(2*before / period)
	log.Printf("Will take %v pictures.", photos)

	for {
		// each day, wait until sunrise
		sr := tomorrowSunrise()
		log.Println("tomorrow sunrise is:", sr)

		// wake up early
		wkup := sr.Add(-before)
		log.Println("wakeup time is:", wkup)

		sltime := wkup.Sub(time.Now())
		log.Println("sleep time:", sltime)
		time.Sleep(sltime)

		timeLapse(photos, period)

		// now wait until sunset
		ss := todaySunset()
		log.Println("today sunset is:", ss)

		// wake up early
		wkup = ss.Add(-before)
		log.Println("wakeup time is:", wkup)

		sltime = wkup.Sub(time.Now())
		log.Println("sleep time:", sltime)
		time.Sleep(sltime)

		timeLapse(photos, period)
	}
}
