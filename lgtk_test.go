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

	var ret []interface{}
	g.WaitExec(func() {
		ret = g.Eval(`return 42`)
	})
	if ret[0].(float64) != 42 {
		t.Fatalf("return not match")
	}

	time.Sleep(time.Second * 1)
}
