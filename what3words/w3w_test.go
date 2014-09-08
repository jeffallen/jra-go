package what3words

import (
	"testing"
)

func TestSearchPosition(t *testing.T) {
	tests := []struct {
		lat, lng float32
		err      error
		want     string
	}{
		{50, 50, nil, "happiness.smuggled.participation"},
		{800, 50, ErrInvalidPosition, "happiness.smuggled.participation"},
	}

	for _, x := range tests {
		got, err := SearchPosition(x.lat, x.lng)
		if err != x.err {
			t.Error("Unexpected error:", err)
			continue
		}
		if x.err != nil {
			continue
		}
		if x.want != got.Text {
			t.Error("For search", x.lat, x.lng, "wanted", x.want, "but got", got)
		}
	}
}
func TestSearchWords(t *testing.T) {
	wordTests := []struct {
		in   string
		err  error
		want []Location
	}{
		{"limit.broom.flip", nil,
			[]Location{{"w3w", 51.52936, -0.151903, "limit.broom.flip", false, ""}}},
		{"*johnoffice", nil,
			[]Location{{"oneword", 51.49826, -0.218548, "*johnoffice", false, ""}}},
		{"dskjfhdsafkja", ErrNotFound, nil},
	}

	for _, x := range wordTests {
		got, err := SearchWords(x.in)
		if err != x.err {
			t.Error("Unexpected error:", err)
			continue
		}
		if x.err != nil {
			continue
		}
		if !compare(x.want, got) {
			t.Error("For search", x.in, "wanted", x.want, "but got", got)
		}
	}
}

func compare(want, got []Location) bool {
	if len(want) != len(got) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
