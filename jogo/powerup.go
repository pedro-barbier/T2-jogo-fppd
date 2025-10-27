package main

import (
	"math/rand"
	"time"
)

func powerUpSpawnar(jogo *Jogo, estrela_obtida chan bool, heal_confirmation chan bool, lock chan struct{}) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	x, y := 0, 0

	// Sorteia uma posição vazia no mapa para spawnar o power-up
	for {
		y = r.Intn(30)
		x = r.Intn(83)
		if jogo.Mapa[y][x] == Vazio {
			<-lock
			jogo.Mapa[y][x] = Powerup
			interfaceDesenharJogo(jogo)
			lock <- struct{}{}
			break
		}
	}
	// Espera até que o power-up seja coletado ou expire após 10 segundos
	select {
	case <-estrela_obtida:
		heal_confirmation <- true
	case <-time.After(10 * time.Second):
		<-lock
		jogo.Mapa[y][x] = Vazio
		interfaceDesenharJogo(jogo)
		lock <- struct{}{}
	}
}
