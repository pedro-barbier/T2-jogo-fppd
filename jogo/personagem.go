// personagem.go - Funções para movimentação e ações do personagem
package main

import (
	"fmt"
	"time"
)

// Atualiza a posição do personagem com base na tecla pressionada (WASD)
func personagemMover(jogo *Jogo, tecla rune, direcao chan (string), estrela_obtida chan bool, lock chan struct{}) {
	dx, dy := 0, 0
	switch tecla {
	case 'w':
		dy = -1 // Move para cima
		<-direcao
		direcao <- "N" // Atualiza a direção para Norte
	case 'a':
		dx = -1 // Move para a esquerda
		<-direcao
		direcao <- "O" // Atualiza a direção para Oeste
	case 's':
		dy = 1 // Move para baixo
		<-direcao
		direcao <- "S" // Atualiza a direção para Sul
	case 'd':
		dx = 1 // Move para a direita
		<-direcao
		direcao <- "L" // Atualiza a direção para Leste
	}

	nx, ny := jogo.PosX+dx, jogo.PosY+dy
	// Verifica se o movimento é permitido e realiza a movimentação
	if jogoPodeMoverPara(jogo, nx, ny) {
		<-lock
		jogoMoverElemento(jogo, jogo.PosX, jogo.PosY, dx, dy)
		jogo.PosX, jogo.PosY = nx, ny
		lock <- struct{}{}
	}
	// Verifica se o personagem andou sobre um power-up
	if jogo.UltimoVisitado == Powerup {
		<-lock
		jogo.UltimoVisitado = Vazio
		jogo.StatusMsg = "Você coletou um power-up! Vida restaurada."
		interfaceDesenharJogo(jogo)
		lock <- struct{}{}
		estrela_obtida <- true
	}
}

// Lógica para o personagem atirar em uma direção
func personagemAtirar(jogo *Jogo, direcao chan (string), lock chan struct{}) {
	<-lock
	jogo.StatusMsg = fmt.Sprintf("Atirando em (%d, %d)", jogo.PosX, jogo.PosY)
	x, y := jogo.PosX, jogo.PosY
	lock <- struct{}{}

	x_ant, y_ant := x, y
	dir := <-direcao
	direcao <- dir

	// Determina a direção do tiro com base na última direção do personagem
	i := 0
	for {
		switch dir {
		case "N":
			y--
			if i > 0 {
				y_ant--
			}
		case "S":
			y++
			if i > 0 {
				y_ant++
			}
		case "L":
			x++
			if i > 0 {
				x_ant++
			}
		case "O":
			x--
			if i > 0 {
				x_ant--
			}
		}

		// Verifica se o tiro atingiu um inimigo
		if jogo.Mapa[y][x] == Inimigo {
			<-lock
			jogo.Mapa[y][x] = Vazio
			if jogo.Mapa[y_ant][x_ant] == Tiro {
				jogo.Mapa[y_ant][x_ant] = Vazio
			}
			interfaceDesenharJogo(jogo)
			lock <- struct{}{}
			break
		} else if jogo.Mapa[y][x] == Vazio { // Move o tiro se o próximo espaço estiver vazio
			<-lock
			jogo.Mapa[y][x] = Tiro
			if jogo.Mapa[y_ant][x_ant] == Tiro {
				jogo.Mapa[y_ant][x_ant] = Vazio
			}
			interfaceDesenharJogo(jogo)
			lock <- struct{}{}

			time.Sleep(100 * time.Millisecond)
		} else { // Tiro atinge uma parede ou outro obstáculo
			<-lock
			jogo.Mapa[y_ant][x_ant] = Vazio
			interfaceDesenharJogo(jogo)
			lock <- struct{}{}
			break
		}
		i++
	}
}

// Processa o evento do teclado e executa a ação correspondente
func personagemExecutarAcao(jogo *Jogo, ev EventoTeclado, direcao chan string, estrela_obtida chan bool, lock chan struct{}) bool {
	switch ev.Tipo {
	case "sair":
		// Retorna false para indicar que o jogo deve terminar
		return false
	case "interagir":
		// Executa a ação de interação
		go personagemAtirar(jogo, direcao, lock)

	case "mover":
		// Move o personagem com base na tecla
		personagemMover(jogo, ev.Tecla, direcao, estrela_obtida, lock)
	}
	return true // Continua o jogo
}
