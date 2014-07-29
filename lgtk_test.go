package lgtk

import (
	"testing"
	"time"
)

func TestLgtk(t *testing.T) {
	g, err := New(`
win = Gtk.Window{
	Gtk.Label{
		label = Text,
	},
}
function win:on_destroy()
	Exit(0)
end
win:show_all()
	`,
		"Text", "Foobarbaz")
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
	time.Sleep(time.Second * 1)
}
