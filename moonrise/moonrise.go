// Package moonrise parses the a moonrise/moonset report
// 
// Feed this package a report from
// http://aa.usno.navy.mil/data/docs/RS_OneYear.php
// and then use its convenient API to query the
// moonrise or moonset for a given day.

package moonrise

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type month struct {
	rise   [31]time.Time
	set    [31]time.Time
	riseok [31]bool
	setok  [31]bool
}

type year struct {
	year   int
	months [12]month
	loc    *time.Location
}

type Db struct {
	years []*year
}

const noyear = 10000

func errFmt(what string) error {
	return fmt.Errorf("problem with the report format: %v", what)
}

func (db *Db) Load(report string) error {
	y, err := read(report)
	if err != nil {
		return err
	}
	db.years = append(db.years, y)
	return nil
}

func (db *Db) Moonrise(tm time.Time) (time.Time, bool) {
	y2 := tm.Year()
	for _, y := range db.years {
		if y.year == y2 {
			return y.Moonrise(tm)
		}
	}
	return zero, false
}

func read(report string) (y *year, err error) {
	y = &year{}

	day := 0
	y.year = noyear
	r := bufio.NewReader(bytes.NewBufferString(report))
	for {
		line, err := r.ReadString('\n')
		if err != nil || day >= 31 {
			break
		}

		if y.year == noyear {
			// Location: W006 24, N46 38                         Rise and Set for the Moon for 2014
			if strings.Contains(line, "for the Moon for ") {
				y64, err := strconv.ParseInt(line[80:84], 10, 32)
				if err != nil {
					return nil, err
				}
				y.year = int(y64)
			}
			continue
		}

		if y.loc == nil {
			if strings.Contains(line, "Zone") {
				// Zone:  1h East of Greenwich
				z := strings.SplitN(line, ":", 2)
				if len(z) != 2 {
					return nil, errFmt("zone line")
				}
				ix := strings.Index(z[1], "h")
				if ix < 0 {
					return nil, errFmt("zone is missing the h")
				}
				hr, err := strconv.ParseFloat(strings.Trim(z[1][0:ix], " "), 64)
				if err != nil {
					return nil, errFmt(err.Error())
				}
				neg := 1
				if strings.Contains(z[1], "West") {
					neg = -1
				}
				y.loc = time.FixedZone("", neg*int(hr*3600))
			}
			continue
		}
		// skip headers:
		//
		// <blank line>
		//        Jan.
		// Day Rise
		//     h m
		if strings.Contains(line, "Jan.") ||
			strings.Contains(line, "Day Rise") ||
			strings.Contains(line, "h m") ||
			len(strings.TrimSpace(line)) == 0 {
			continue
		}

		if len(line) < (4 + 12*11) {
			return nil, errFmt("day line too short")
		}

		line = strings.TrimLeft(line, "\t")

		// 01  0844 1822  0921      <- 4 spaces mark no event
		for mon, pt := 0, 5; mon < 12; mon++ {
			rise := line[pt : pt+4]
			set := line[pt+5 : pt+9]
			pt += 11

			if rise == "    " {
				y.months[mon].riseok[day] = false
			} else {
				h, _ := strconv.ParseInt(rise[0:2], 10, 32)
				m, _ := strconv.ParseInt(rise[2:], 10, 32)
				r := time.Date(y.year, time.Month(mon), day, int(h), int(m), 0, 0, y.loc)
				y.months[mon].rise[day] = r
				y.months[mon].riseok[day] = true
			}
			if set == "    " {
				y.months[mon].setok[day] = false
			} else {
				h, _ := strconv.ParseInt(set[0:2], 10, 32)
				m, _ := strconv.ParseInt(set[2:], 10, 32)
				r := time.Date(y.year, time.Month(mon), day, int(h), int(m), 0, 0, y.loc)
				y.months[mon].set[day] = r
				y.months[mon].setok[day] = true
			}
		}
		day++
	}
	return y, nil
}

var zero time.Time

func (y *year) Moonrise(tm time.Time) (time.Time, bool) {
	if tm.Year() != y.year {
		return zero, false
	}
	return y.months[tm.Month()-1].rise[tm.Day()-1],
		y.months[tm.Month()-1].riseok[tm.Day()-1]
}
