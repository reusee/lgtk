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
		id = 'label',
	},
}
function win:on_destroy()
	Exit(0)
end

function set_label(s)
	win.child.label:set_label(s)
end
function get_label()
	return win.child.label:get_label()
end

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

	g.Exec(func() {
		g.Call("set_label", ">>> "+g.Call("get_label")[0].(string))
	})

	g.ExecEval(`win:show_all()`)

	time.Sleep(time.Second * 1)
}

func TestClose(t *testing.T) {
	g, err := New(``)
	if err != nil {
		t.Fatal(err)
	}
	g.Close()
}
