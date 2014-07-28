package lgtk

import (
	"testing"
	"time"
)

func TestLgtk(t *testing.T) {
	g, err := New(`
win = Gtk.Window{}
function win:on_destroy()
	Exit(0)
end
win:show_all()
	`, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	g.Exec(`
Return('foobar')
	`)
	ret := (<-g.Return).(string)
	if ret != "foobar" {
		t.Fatalf("return not match")
	}
	time.Sleep(time.Second * 3)
}
