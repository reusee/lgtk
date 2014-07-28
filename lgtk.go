package lgtk

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/reusee/lgo"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Gtk struct {
	*lgo.Lua
	Return     chan interface{}
	codeToExec chan string
	conn       net.Conn
}

func New(code string) (*Gtk, error) {
	// init
	g := &Gtk{
		Lua:        lgo.NewLua(),
		Return:     make(chan interface{}, 8),
		codeToExec: make(chan string, 8),
	}

	// functions
	g.Lua.RegisterFunctions(map[string]interface{}{
		"Exit": func(i int) {
			os.Exit(i)
		},
		"Return": func(v interface{}) {
			g.Return <- v
		},
	})

	// eval notify
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(30000+rand.Intn(20000)))
	if err != nil {
		return nil, err
	}
	luaConnected := make(chan bool)
	var acceptErr error
	go func() {
		g.conn, acceptErr = ln.Accept()
		close(luaConnected)
	}()
	g.Lua.RegisterFunction("_Exec", func() {
		g.Lua.RunString(<-g.codeToExec)
	})

	// start lua
	g.RunString(`
lgi = require('lgi')
Gtk = lgi.require('Gtk', '3.0')
Gio = lgi.Gio
GLib = lgi.GLib

socket = Gio.Socket.new(Gio.SocketFamily.IPV4, Gio.SocketType.STREAM, Gio.SocketProtocol.TCP)
	`)
	host, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		return nil, err
	}
	g.RunString(fmt.Sprintf(`
socket:connect(Gio.InetSocketAddress.new_from_string("%s", %s))
	`, host, port))
	g.RunString(`
channel = GLib.IOChannel.unix_new(socket.fd)
bytes = require('bytes')
buf = bytes.new(1)
GLib.io_add_watch(channel, GLib.PRIORITY_DEFAULT, GLib.IOCondition.IN, function()
	_Exec()
	socket:receive(buf)
	return true
end)
	`)
	g.RunString(code)
	go g.RunString("Gtk.main()")

	// wait lua
	select {
	case <-luaConnected:
	case <-time.After(time.Second):
		return nil, errors.New("lua not connecting")
	}

	return g, nil
}

func (g *Gtk) Exec(code string) {
	g.codeToExec <- code
	g.conn.Write([]byte{'_'})
}
