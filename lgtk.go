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

type Gtk struct {
	*lua.Lua
	queue chan func()
	conn  net.Conn
}

func New(code string, bindings ...interface{}) (*Gtk, error) {
	// init
	l, err := lua.New()
	if err != nil {
		return nil, err
	}
	g := &Gtk{
		Lua:   l,
		queue: make(chan func(), 8),
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
	_Exec()
	socket:receive(buf)
	return true
end)
	`)
	g.Eval(code)
	go g.Eval("Gtk.main()")

	// wait lua
	select {
	case <-luaConnected:
	case <-time.After(time.Second):
		return nil, errors.New("lua not connecting")
	}

	return g, nil
}

func (g *Gtk) Exec(fun func()) {
	g.queue <- fun
	g.conn.Write([]byte{'_'})
}

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

func (g *Gtk) ExecEval(code string, envs ...interface{}) {
	g.Exec(func() {
		g.Eval(code, envs...)
	})
}
