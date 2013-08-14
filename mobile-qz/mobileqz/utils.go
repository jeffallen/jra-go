package mobileqz

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func fixHeadline(in string) string {
	words := strings.Split(in, " ")
	if len(words) < 2 {
		return in
	}
	first := words[0]

	// Here's why this headline writer needs to be fired
	if first == "Here’s" ||
		first == "Here&#8217;s" ||
		first == "Here's" {
		words = words[1:]
		words[0] = titleCaseFirstLetter(words[0])
		// do it again to catch the worst case scenario:
		// Here's 5 reasons headlines like this suck!
		return fixHeadline(strings.Join(words, " "))
	}

	// 5 reasons headlines like this suck
	if isSmallNumber(first) && isPlural(words[1]) {
		words = words[2:]
		words[0] = titleCaseFirstLetter(words[0])
		return strings.Join(words, " ")
	}

	return in
}

func isPlural(in string) bool {
	return strings.HasSuffix(in, "s")
}

func isSmallNumber(in string) bool {
	in = strings.ToLower(in)
	switch in {
	case "2", "3", "4", "5", "6", "7", "8", "9", "10",
		"two", "three", "four", "five", "six", "seven", "eight", "nine", "ten":
		return true
	}
	return false
}

func titleCaseFirstLetter(in string) string {
	out := ""
	out += string(in[0])
	out = strings.ToTitle(out)
	out += in[1:]
	return out
}

var reFixImg = regexp.MustCompile(`w=[0-9]+&#038;h=[0-9]+`)
var reFixImg2 = regexp.MustCompile(`width="[0-9]+" height="[0-9]+"`)
var reNum = regexp.MustCompile(`([0-9]+)`)

func fixImages(in string, width int) string {
	out := reFixImg.ReplaceAllFunc([]byte(in), func(in []byte) []byte {
		wh := reNum.FindAll(in, 3)
		if len(wh) == 3 {
			w, _ := strconv.Atoi(string(wh[0]))
			// wh[1] is 038 because of the stupid HTML entity
			h, _ := strconv.Atoi(string(wh[2]))
			return []byte(fmt.Sprintf("w=%d&#038;h=%d",
				width, h/(w/width)))
		}
		return in
	})
	out = reFixImg2.ReplaceAllFunc(out, func(in []byte) []byte {
		wh := reNum.FindAll(in, 2)
		if len(wh) == 2 {
			w, _ := strconv.Atoi(string(wh[0]))
			h, _ := strconv.Atoi(string(wh[1]))
			// avoid div by zero
			if width == 0 || w/width == 0 {
				return in
			}
			return []byte(fmt.Sprintf("width=\"%d\" height=\"%d\"",
				width, h/(w/width)))
		}
		return in
	})
	return string(out)
}

var reFixQzLinks = regexp.MustCompile(`http://qz.com/([0-9]+)/[^/]*/`)

func fixQzLinks(in string) string {
	out := reFixQzLinks.ReplaceAllFunc([]byte(in), func(in []byte) []byte {
		nums := reNum.FindAll(in, 3)
		if len(nums) >= 1 {
			return []byte("/article/" + string(nums[0]))
		}
		return in
	})
	return string(out)
}
