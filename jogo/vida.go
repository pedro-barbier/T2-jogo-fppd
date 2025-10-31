package main

func vidaAdm(jogo *Jogo, heal_confirmation chan bool, damage_confirmation chan bool, gameOver chan bool, lock chan struct{}) {
	vida := 3

	// Loop principal de administração da vida do personagem
	for {
		select {
		case cura := <-heal_confirmation:
			// Cura o personagem se ele não estiver com vida cheia
			if cura && vida < 3 {
				vida++
				switch vida {
				case 3:
					<-lock
					jogo.Mapa[2][4] = Coracao
					jogo.StatusMsg = "Vida cheia!"
					interfaceDesenharJogo(jogo)
					lock <- struct{}{}
				case 2:
					<-lock
					jogo.Mapa[2][6] = Coracao
					jogo.StatusMsg = "Você recuperou vida!"
					interfaceDesenharJogo(jogo)
					lock <- struct{}{}
				}
				// Notifica servidor: curou (atualiza vidas)
				rpcEmitAction("heal", vida)
			}

		case dano := <-damage_confirmation:
			// Aplica dano ao personagem e verifica se ele morreu
			if dano {
				vida--
				switch vida {
				case 2:
					<-lock
					jogo.Mapa[2][4] = CoracaoFerido
					jogo.StatusMsg = "Você tomou dano!"
					interfaceDesenharJogo(jogo)
					lock <- struct{}{}
				case 1:
					<-lock
					jogo.Mapa[2][6] = CoracaoFerido
					jogo.StatusMsg = "Você tomou dano!"
					interfaceDesenharJogo(jogo)
					lock <- struct{}{}
				case 0:
					<-lock
					jogo.Mapa[2][8] = CoracaoFerido
					jogo.StatusMsg = "Você morreu! Fim de jogo."
					interfaceDesenharJogo(jogo)
					lock <- struct{}{}
					// Notifica servidor: dano (vidas foram a 0)
					rpcEmitAction("damaged", vida)
					gameOver <- true
					return
				}
				// Notifica servidor: tomou dano (atualiza vidas)
				rpcEmitAction("damaged", vida)
			}
		}
	}
}
