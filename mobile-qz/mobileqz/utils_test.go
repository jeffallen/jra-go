package mobileqz

import (
	"testing"
)

var headlineTests = []struct {
	input  string
	output string
}{
	{"Here’s what’s behind the Chinese cash crunch",
		"What’s behind the Chinese cash crunch"},
	{"Here&#8217;s what&#8217;s behind the Chinese cash crunch",
		"What&#8217;s behind the Chinese cash crunch"},
	{"5 reasons headlines like this suck",
		"Headlines like this suck"},
}

func TestHeadlineFix(t *testing.T) {
	for _, x := range headlineTests {
		o := fixHeadline(x.input)
		if x.output != o {
			t.Error("Got", o)
		}
	}
}

func TestImageFix(t *testing.T) {
	in := `
<img src="http://qzprod.files.wordpress.com/2013/06/screen-shot-2013-06-18-at-4-29-21-pm.png?w=640&#038;h=286" width="640" height="286" class="size-medium_10" data-retina="http://qzprod.files.wordpress.com/2013/06/screen-shot-2013-06-18-at-4-29-21-pm.png?w=1064" alt="" title=""/>
<img src="http://qzprod.files.wordpress.com/2013/06/screen-shot-2013-06-18-at-4-29-21-pm.png?w=640&#038;h=286" width="640" height="286" class="size-medium_10" data-retina="http://qzprod.files.wordpress.com/2013/06/screen-shot-2013-06-18-at-4-29-21-pm.png?w=1064" alt="" title=""/>
`
	exp := `
<img src="http://qzprod.files.wordpress.com/2013/06/screen-shot-2013-06-18-at-4-29-21-pm.png?w=320&#038;h=143" width="320" height="143" class="size-medium_10" data-retina="http://qzprod.files.wordpress.com/2013/06/screen-shot-2013-06-18-at-4-29-21-pm.png?w=1064" alt="" title=""/>
<img src="http://qzprod.files.wordpress.com/2013/06/screen-shot-2013-06-18-at-4-29-21-pm.png?w=320&#038;h=143" width="320" height="143" class="size-medium_10" data-retina="http://qzprod.files.wordpress.com/2013/06/screen-shot-2013-06-18-at-4-29-21-pm.png?w=1064" alt="" title=""/>
`
	out := fixImages(in, 320)
	if out != exp {
		t.Error("Got", out)
	}
}
