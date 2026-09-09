package mqttx

import "testing"

func TestTopicsAreFarmScoped(t *testing.T) {
	if got, want := PingTopic("f1", "cow-07"), "farm/f1/collar/cow-07/ping"; got != want {
		t.Errorf("PingTopic = %q, want %q", got, want)
	}
	if got, want := PingPattern("f1"), "farm/f1/collar/+/ping"; got != want {
		t.Errorf("PingPattern = %q, want %q", got, want)
	}
	if got, want := FenceTopic("f1", "p1"), "farm/f1/fence/p1"; got != want {
		t.Errorf("FenceTopic = %q, want %q", got, want)
	}
}

// TestFarmFromTopic is the wall between farms. Anything that is not exactly
// one farm's ping topic must be refused, not guessed at — this is the
// function Loop 5's multi-tenancy leans on.
func TestFarmFromTopic(t *testing.T) {
	cases := []struct {
		topic string
		farm  string
		ok    bool
	}{
		{"farm/f1/collar/cow-07/ping", "f1", true},
		{"farm/f2/collar/c1/ping", "f2", true},
		{"farm/f1/collar/cow-07/status", "", false},
		{"farm/f1/fence/p1", "", false},
		{"farm/f1/collar/cow-07/ping/extra", "", false},
		{"collar/cow-07/ping", "", false},
		{"", "", false},
	}

	for _, c := range cases {
		farm, ok := farmFromTopic(c.topic)
		if ok != c.ok || farm != c.farm {
			t.Errorf("farmFromTopic(%q) = %q, %v; want %q, %v", c.topic, farm, ok, c.farm, c.ok)
		}
	}
}
