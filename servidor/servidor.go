package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
)

// Tipos básicos
type PlayerID string

type Player struct {
	ID        PlayerID
	Name      string
	X, Y      int
	Lives     int
	Dir       string
	LastSeen  time.Time
	Connected bool
}

type PlayerSummary struct {
	ID    PlayerID
	Name  string
	X, Y  int
	Lives int
	Dir   string
}

// Eventos enviados aos clientes para atualizarem o estado localmente
type Event struct {
	ID   uint64 // sequencial
	Type string // "player_join" | "player_leave" | "player_state"
	// Dados comuns
	Player PlayerSummary
	// Dados auxiliares
	Text string
}

// RPC: argumentos e respostas
type JoinArgs struct {
	Name string
}
type JoinReply struct {
	ID      PlayerID
	Self    PlayerSummary
	Players []PlayerSummary // estado atual dos demais
	Since   uint64          // último EventID conhecido
}

type LeaveArgs struct {
	ID PlayerID
}
type LeaveReply struct{}

type UpdateArgs struct {
	ID     PlayerID
	X, Y   int
	Lives  int
	Dir    string
	Action string // "move", "shoot", "heal", "hit"...
}
type UpdateReply struct {
	EventID uint64
}

type PollArgs struct {
	Since uint64
}
type PollReply struct {
	Events []Event
}

type GameServer struct {
	mu          sync.Mutex
	players     map[PlayerID]*Player
	nextID      int
	nextEventID uint64
	events      []Event
}

func NewGameServer() *GameServer {
	return &GameServer{
		players:     make(map[PlayerID]*Player),
		nextID:      1,
		nextEventID: 0,
		events:      make([]Event, 0, 1024),
	}
}

func (s *GameServer) nextPlayerID() PlayerID {
	s.nextID++
	return PlayerID(fmt.Sprintf("p%03d", s.nextID))
}

func (s *GameServer) appendEvent(ev Event) uint64 {
	s.nextEventID++
	ev.ID = s.nextEventID
	s.events = append(s.events, ev)
	if !(ev.Type == "player_state" && ev.Text == "tick") {
		log.Printf("event %d type=%s player=%s(%s) text=%s", ev.ID, ev.Type, ev.Player.Name, ev.Player.ID, ev.Text)
	}
	return ev.ID
}

func (s *GameServer) Join(args JoinArgs, reply *JoinReply) error {
	if args.Name == "" {
		args.Name = "Player"
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextPlayerID()
	p := &Player{
		ID:        id,
		Name:      args.Name,
		X:         40,
		Y:         14,
		Lives:     3,
		Dir:       "S",
		LastSeen:  time.Now(),
		Connected: true,
	}
	s.players[id] = p

	// Monta lista dos demais
	others := make([]PlayerSummary, 0, len(s.players)-1)
	for pid, pl := range s.players {
		if pid == id {
			continue
		}
		others = append(others, PlayerSummary{
			ID:    pl.ID,
			Name:  pl.Name,
			X:     pl.X,
			Y:     pl.Y,
			Lives: pl.Lives,
			Dir:   pl.Dir,
		})
	}

	// Evento de entrad
	ev := Event{
		Type: "player_join",
		Player: PlayerSummary{
			ID:    p.ID,
			Name:  p.Name,
			X:     p.X,
			Y:     p.Y,
			Lives: p.Lives,
			Dir:   p.Dir,
		},
		Text: fmt.Sprintf("%s entrou no jogo", p.Name),
	}
	lastID := s.appendEvent(ev)

	reply.ID = id
	reply.Self = PlayerSummary{
		ID:    p.ID,
		Name:  p.Name,
		X:     p.X,
		Y:     p.Y,
		Lives: p.Lives,
		Dir:   p.Dir,
	}
	reply.Players = others
	reply.Since = lastID
	return nil
}

// Leave remove jogador
func (s *GameServer) Leave(args LeaveArgs, reply *LeaveReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pl, ok := s.players[args.ID]
	if !ok {
		return nil
	}
	delete(s.players, args.ID)

	ev := Event{
		Type: "player_leave",
		Player: PlayerSummary{
			ID:    pl.ID,
			Name:  pl.Name,
			X:     pl.X,
			Y:     pl.Y,
			Lives: pl.Lives,
			Dir:   pl.Dir,
		},
		Text: fmt.Sprintf("%s saiu", pl.Name),
	}
	s.appendEvent(ev)
	return nil
}

// Update registra posição/vidas/direção do jogador e emite player_state
func (s *GameServer) Update(args UpdateArgs, reply *UpdateReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pl, ok := s.players[args.ID]
	if !ok {
		return errors.New("player not found")
	}
	// detect previous position/vida para gerar logs de movimento
	prevX, prevY := pl.X, pl.Y
	prevLives := pl.Lives

	// aplica atualização
	pl.X = args.X
	pl.Y = args.Y
	if args.Lives > 0 {
		pl.Lives = args.Lives
	}
	if args.Dir != "" {
		pl.Dir = args.Dir
	}
	pl.LastSeen = time.Now()

	// Decide o texto do evento: prefere actions explícitas, mas marca movimento quando a posição mudou.
	actionText := args.Action
	if actionText == "" || actionText == "tick" {
		if prevX != pl.X || prevY != pl.Y {
			actionText = fmt.Sprintf("moved %d,%d", pl.X, pl.Y)
		} else if actionText == "" {
			actionText = ""
		}
	}

	// Também loga mudanças de vida como eventos legíveis
	if prevLives != pl.Lives {
		if actionText != "" {
			actionText = fmt.Sprintf("%s; lives=%d", actionText, pl.Lives)
		} else {
			actionText = fmt.Sprintf("lives=%d", pl.Lives)
		}
	}

	ev := Event{
		Type: "player_state",
		Player: PlayerSummary{
			ID:    pl.ID,
			Name:  pl.Name,
			X:     pl.X,
			Y:     pl.Y,
			Lives: pl.Lives,
			Dir:   pl.Dir,
		},
		Text: actionText,
	}
	reply.EventID = s.appendEvent(ev)
	return nil
}

// Poll retorna eventos desde um ID
func (s *GameServer) Poll(args PollArgs, reply *PollReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if args.Since >= s.nextEventID {
		reply.Events = nil
		return nil
	}
	start := 0
	for i := len(s.events) - 1; i >= 0; i-- {
		if s.events[i].ID <= args.Since {
			start = i + 1
			break
		}
	}
	max := start + 256
	if max > len(s.events) {
		max = len(s.events)
	}
	reply.Events = append([]Event(nil), s.events[start:max]...)
	return nil
}

func main() {
	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = ":12345"
	}
	srv := NewGameServer()

	if err := rpc.RegisterName("GameServer", srv); err != nil {
		log.Fatalf("rpc.Register: %v", err)
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}
	log.Printf("Servidor RPC ouvindo em %s", addr)
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go rpc.ServeConn(conn)
	}
}
