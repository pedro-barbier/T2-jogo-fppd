package main

import (
	"math/rand"
	"time"
)

func inimigoSpawnar(jogo *Jogo, damage_confirmation chan bool, lock chan struct{}) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	side := r.Intn(4)

	pos_spawns_y, pos_spawns_x := 0, 0
	x, y := 0, 0

	// Define a posição inicial do inimigo com base no lado e posição sorteado
spawnLoop:
	for {
		pos_spawns_y = r.Intn(4) + 13
		pos_spawns_x = r.Intn(16) + 35
		switch side {
		case 0: // Top
			if jogo.Mapa[1][pos_spawns_x] == Vazio {
				<-lock
				jogo.Mapa[1][pos_spawns_x] = Inimigo
				interfaceDesenharJogo(jogo)
				lock <- struct{}{}
				y = 1
				x = pos_spawns_x
				break spawnLoop
			}
		case 1: // Right
			if jogo.Mapa[pos_spawns_y][80] == Vazio {
				<-lock
				jogo.Mapa[pos_spawns_y][80] = Inimigo
				interfaceDesenharJogo(jogo)
				lock <- struct{}{}
				y = pos_spawns_y
				x = 80
				break spawnLoop
			}
		case 2: // Bottom
			if jogo.Mapa[28][pos_spawns_x] == Vazio {
				<-lock
				jogo.Mapa[28][pos_spawns_x] = Inimigo
				interfaceDesenharJogo(jogo)
				lock <- struct{}{}
				y = 28
				x = pos_spawns_x
				break spawnLoop
			}
		case 3: // Left
			if jogo.Mapa[pos_spawns_y][1] == Vazio {
				<-lock
				jogo.Mapa[pos_spawns_y][1] = Inimigo
				interfaceDesenharJogo(jogo)
				lock <- struct{}{}
				y = pos_spawns_y
				x = 1
				break spawnLoop
			}
		}
	}

	// Lógica de movimentação do inimigo em direção ao personagem
	for {
		px, py := jogo.PosX, jogo.PosY

		time.Sleep(500 * time.Millisecond)

		// Calcula a direção do movimento em direção ao personagem
		dx, dy := 0, 0
		if px > x && jogo.Mapa[y][x+1] != Parede && jogo.Mapa[y][x-1] != Inimigo && jogo.Mapa[y][x-1] != Tiro {
			dx = 1
		} else if px < x && jogo.Mapa[y][x-1] != Parede && jogo.Mapa[y][x-1] != Inimigo && jogo.Mapa[y][x-1] != Tiro {
			dx = -1
		} else if py > y && jogo.Mapa[y+1][x] != Parede && jogo.Mapa[y][x-1] != Inimigo && jogo.Mapa[y][x-1] != Tiro {
			dy = 1
		} else if py < y && jogo.Mapa[y-1][x] != Parede && jogo.Mapa[y][x-1] != Inimigo && jogo.Mapa[y][x-1] != Tiro {
			dy = -1
		}
		if jogo.Mapa[y][x] != Inimigo {
			break
		}

		// Verifica se o inimigo atingiu o personagem
		if y+dy == py && x+dx == px {
			<-lock
			jogo.Mapa[y][x] = Vazio
			// jogo.StatusMsg = "Você foi atingido!"
			interfaceDesenharJogo(jogo)
			damage_confirmation <- true
			lock <- struct{}{}
			break
		}

		// Move o inimigo para a nova posição
		<-lock
		jogo.Mapa[y+dy][x+dx] = Inimigo
		jogo.Mapa[y][x] = Vazio
		interfaceDesenharJogo(jogo)
		lock <- struct{}{}

		// Atualiza a posição atual do inimigo
		y += dy
		x += dx
	}
}
