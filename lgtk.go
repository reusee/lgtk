package lgtk

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/reusee/lua"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Gtk is a lua vm instance including Gtk helpers
type Gtk struct {
	*lua.Lua
	queue    chan func()
	conn     net.Conn
	MainQuit chan struct{}
}

// New creates a new Gtk struct
func New(code string, bindings ...interface{}) (*Gtk, error) {
	// init
	l, err := lua.New()
	if err != nil {
		return nil, err
	}
	g := &Gtk{
		Lua:      l,
		queue:    make(chan func(), 8),
		MainQuit: make(chan struct{}),
	}

	// functions
	g.Lua.Set(
		"Exit", func(i int) {
			os.Exit(i)
		},
	)
	err = g.Lua.Pset(bindings...)
	if err != nil {
		return nil, err
	}

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
	g.Lua.Set("_Exec", func() {
		(<-g.queue)()
	})

	// start lua
	g.Eval(`
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
	g.Eval(fmt.Sprintf(`
socket:connect(Gio.InetSocketAddress.new_from_string("%s", %s))
	`, host, port))
	g.Eval(`
channel = GLib.IOChannel.unix_new(socket.fd)
bytes = require('bytes')
buf = bytes.new(1)
GLib.io_add_watch(channel, GLib.PRIORITY_DEFAULT, GLib.IOCondition.IN, function()
	socket:receive(buf)
	_Exec()
	return true
end)
	`)
	g.Eval(code)

	// main
	g.Set("_SigMainQuit", func() {
		close(g.MainQuit)
	})
	go g.Eval(`
Gtk.main()
_SigMainQuit()
	`)

	// wait lua
	select {
	case <-luaConnected:
	case <-time.After(time.Second):
		return nil, errors.New("lua not connecting")
	}

	return g, nil
}

// Exec runs a function thread-safely and asynchronously
func (g *Gtk) Exec(fun func()) {
	g.queue <- fun
	g.conn.Write([]byte{'_'})
}

// WaitExec runs a function thread-safely and wait for execution done
func (g *Gtk) WaitExec(fun func()) {
	var m sync.Mutex
	m.Lock()
	g.queue <- func() {
		fun()
		m.Unlock()
	}
	g.conn.Write([]byte{'_'})
	m.Lock()
}

// ExecEval runs a piece of code thread-safely and asynchronously
func (g *Gtk) ExecEval(code string, envs ...interface{}) {
	g.Exec(func() {
		g.Eval(code, envs...)
	})
}

// Close runs gtk_main_quit and closes the lua vm
func (g *Gtk) Close() {
	g.ExecEval(`Gtk.main_quit()`)
	<-g.MainQuit
	g.Lua.Close()
}
