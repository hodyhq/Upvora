package ratelimit_test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/ratelimit"
)

func TestWindow_AllowsUpToMaxPerKey(t *testing.T) {
	RegisterT(t)
	w := ratelimit.New(2, time.Hour)
	Expect(w.Allow("a")).IsTrue()
	Expect(w.Allow("a")).IsTrue()
	Expect(w.Allow("a")).IsFalse()
	Expect(w.Allow("b")).IsTrue()
}

func TestWindow_Expires(t *testing.T) {
	RegisterT(t)
	w := ratelimit.New(1, 20*time.Millisecond)
	Expect(w.Allow("a")).IsTrue()
	Expect(w.Allow("a")).IsFalse()
	time.Sleep(30 * time.Millisecond)
	Expect(w.Allow("a")).IsTrue()
}

func TestWindow_BoundedKeys(t *testing.T) {
	RegisterT(t)
	w := ratelimit.New(1, 20*time.Millisecond)
	for i := 0; i < 50000; i++ {
		w.Allow(fmt.Sprintf("k%d", i))
	}
	time.Sleep(30 * time.Millisecond)
	w.Allow("trigger-sweep")
	Expect(w.Keys() <= 10001).IsTrue()
}
