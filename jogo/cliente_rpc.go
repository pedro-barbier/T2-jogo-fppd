package main

import (
	"log"
	"net/rpc"
	"os"
	"time"
)

// Estruturas replicadas do servidr
type rpcPlayerID string

type rpcPlayerSummary struct {
	ID    rpcPlayerID
	Name  string
	X, Y  int
	Lives int
	Dir   string
}
type rpcEvent struct {
	ID     uint64
	Type   string // "player_join" | "player_leave" | "player_state"
	Player rpcPlayerSummary
	Text   string
}

type rpcJoinArgs struct{ Name string }
type rpcJoinReply struct {
	ID      rpcPlayerID
	Self    rpcPlayerSummary
	Players []rpcPlayerSummary
	Since   uint64
}
type rpcLeaveArgs struct{ ID rpcPlayerID }
type rpcLeaveReply struct{}
type rpcUpdateArgs struct {
	ID     rpcPlayerID
	X, Y   int
	Lives  int
	Dir    string
	Action string
}
type rpcUpdateReply struct{ EventID uint64 }
type rpcPollArgs struct{ Since uint64 }
type rpcPollReply struct{ Events []rpcEvent }

type rpcClientState struct {
	client   *rpc.Client
	selfID   rpcPlayerID
	lastEvID uint64
	others   map[rpcPlayerID]struct {
		X int
		Y int
	}
}

// rpcEmitAction serve para notificar o servidor sobre ações locais.
var rpcEmitAction func(action string, lives int)

func init() {
	rpcEmitAction = func(string, int) {}
}

// Inicia cliente RPC
func startRPCClient(jogo *Jogo, lock chan struct{}) {
	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		return
	}
	name := os.Getenv("PLAYER_NAME")
	if name == "" {
		name = "Player"
	}

	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		log.Printf("RPC: dial falhou: %v", err)
		return
	}
	st := &rpcClientState{
		client: client,
		others: make(map[rpcPlayerID]struct{ X, Y int }),
	}

	// Configura o gancho para emissão de ações (atirar, acertar, curar, dano...)
	rpcEmitAction = func(action string, lives int) {
		// Captura coordenadas atuais do jogador
		x, y := jogo.PosX, jogo.PosY
		req := rpcUpdateArgs{
			ID:     st.selfID,
			X:      x,
			Y:      y,
			Lives:  lives, // 0 = não altera
			Dir:    "",
			Action: action,
		}
		var rep rpcUpdateReply
		go func() { _ = st.client.Call("GameServer.Update", req, &rep) }()
	}

	// Join
	var joinRep rpcJoinReply
	if err := client.Call("GameServer.Join", rpcJoinArgs{Name: name}, &joinRep); err != nil {
		log.Printf("RPC: Join falhou: %v", err)
		return
	}
	st.selfID = joinRep.ID
	st.lastEvID = joinRep.Since

	// Desenha jogadores existentes
	for _, p := range joinRep.Players {
		st.others[p.ID] = struct{ X, Y int }{p.X, p.Y}
		<-lock
		jogo.Mapa[p.Y][p.X] = SegundoJogador
		interfaceDesenharJogo(jogo)
		lock <- struct{}{}
	}

	// Loop de atualização
	go func() {
		tick := time.NewTicker(200 * time.Millisecond)
		defer tick.Stop()
		for range tick.C {
			// Envia estado atual do jogador local
			upReq := rpcUpdateArgs{
				ID:     st.selfID,
				X:      jogo.PosX,
				Y:      jogo.PosY,
				Lives:  0, // 0 = não alterar no servidor
				Dir:    "",
				Action: "tick",
			}
			var upRep rpcUpdateReply
			_ = st.client.Call("GameServer.Update", upReq, &upRep)

			// Busca eventos novos
			var pr rpcPollReply
			if err := st.client.Call("GameServer.Poll", rpcPollArgs{Since: st.lastEvID}, &pr); err != nil {
				continue
			}
			for _, ev := range pr.Events {
				st.lastEvID = ev.ID
				switch ev.Type {
				case "player_join", "player_state":
					// Atualiza/Desenha outros jogadores
					if ev.Player.ID == st.selfID {
						continue
					}
					prev, ok := st.others[ev.Player.ID]
					if ok {
						<-lock
						// Limpa posição antiga se ainda marcada como SegundoJogador
						if jogo.Mapa[prev.Y][prev.X] == SegundoJogador {
							jogo.Mapa[prev.Y][prev.X] = Vazio
						}
						lock <- struct{}{}
					}
					st.others[ev.Player.ID] = struct{ X, Y int }{ev.Player.X, ev.Player.Y}
					<-lock
					jogo.Mapa[ev.Player.Y][ev.Player.X] = SegundoJogador
					interfaceDesenharJogo(jogo)
					lock <- struct{}{}
				case "player_leave":
					// Remove do mapa
					prev, ok := st.others[ev.Player.ID]
					if ok {
						<-lock
						if jogo.Mapa[prev.Y][prev.X] == SegundoJogador {
							jogo.Mapa[prev.Y][prev.X] = Vazio
							interfaceDesenharJogo(jogo)
						}
						lock <- struct{}{}
						delete(st.others, ev.Player.ID)
					}
				}
			}
		}
	}()
}
