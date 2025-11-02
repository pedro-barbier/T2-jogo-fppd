Projeto feito por Pedro Barbieri, Guilherme Cavazzotto, Bernardo Fensterseifer e Gabriel Costa a partir de um projeto por Marcelo Veiga Neves

## Jogo de Terminal em Go

## Como compilar

- Instalar go

### 1) Rodar o servidor (terminal 1)

```bash
cd servidor
SERVER_ADDR=":12345" go run .
```

### 2) Rodar o jogo (terminal 2 e 3)

```bash
cd jogo
SERVER_ADDR="127.0.0.1:12345" PLAYER_NAME="Jogador" go run .
```

Controles: WASD para mover, E para atirar, ESC para sair.

### (Opcional) Compilar binários

```bash
# Servidor
cd servidor
go build -o servidor .
SERVER_ADDR=":12345" ./servidor

# Jogo
cd ../jogo
go build -o jogo .
SERVER_ADDR="127.0.0.1:12345" ./jogo
```

# Relatório T1:

# 3+ novos elementos concorrentes

- Ação de disparar: A função `personagemInteragir()` foi alterada para `personagemAtirar()`. Sempre que `personagemExecutarAcao()` for chamado e o usuário tiver clicado `e`, `personagemAtirar()` será chamado concorrentemente criando uma nova rotina go, fazendo com que um caractere simbolizando um tiro traverse o mapa até atingir um inimigo ou uma parede, apagando um inimigo caso encoste em um, e apenas sumindo no caso de atingir uma parede ou vegetação.
- Geração de estrelas: Em `main.go`, antes de entrar no loop de movimentação e ação do jogador, temos uma rotina go `powerUpSpawnar()`que gera estrelas em coordenadas aleatórias pelo mapa a cada 15 segundos. Elas somem após 10 segundos caso o jogador não às pegue a tempo. As estrelas servirão para aplicação da administração da vida através da função `admVIda()` do jogador, que ouvirá múltiplos canais para saber quando o jogador encostou em um inimigo e sofreu dano, e quando ele pegou uma estrela e deve ser curado.
- Geração de inimigos: Assim como na geração de estrelas, `main.go`, antes de entrar no loop de movimentação e ação do jogador, temos uma rotina go `inimigoSpawnar()` que gera inimigos nos extremos cardinais do mapa, e caso consiga colocar ele no mapa, começa um loop de movimento tentando igualar sua coordenada com a do jogador, até chegar nele, andando apenas em espaços diferentes de uma parede. Caso o inimigo consiga chegar até o jogador sem ser removido por um tiro antes, como a rotina de estrelas, informa através de um canal para `admVida()` o ocorrido, diminuindo do jogador um coração de vida.

# Exclusão mútua

Para este projeto, como foi proibido o uso de mutex e outros artefatos de evitar condições de corrida, em `main.go` foi criado um canal `lock`, de elementos `struct{}` que serão utilizados como tokens, inicialmente com um token, para evitar condições de corrida. Sempre que uma variável utilizada por várias rotinas diferentes deve ser alterada, ou quando a interface deve ser desenhada, tiramos o token do canal lock, e apenas devolvemos o token quando a ação for realizada. Isso evita que outras rotinas alterem a váriavel ou desenhem o jogo ao mesmo tempo, pois cada uma deve esperar o token ser devolvido, eliminando condição de corrida.

# Comunicação entre canais

Isso ocorre em diversas parte do projeto desenvolvido, e quase em todos as go routines aplicadas até o momento. Aqui estão alguns exemplos

- Ação de disparar: `personagemAtirar()` e `personagemMover()` recebem da main através de `personagemExecutarAcao()` um canal `direcao`, que é utilizado para informar ao atirar qual a direção que o disparo deve seguir. Sempre quando o personagem se move para alguma direção, o canal recebe uma string de uma letra, sendo ela "N", "L", "S" ou "O", indicando a última direção para qual o personagem se moveu. A rotina de disparo por sua vez lê a última direção que o personagem se moveu através do canal e a partir disso decide para onde o tiro deve ir.
- Inimigos e estrelas: As rotinas de geração de ambos recebem canais diferentes para informar a rotina de administração de vida quando um inimigo entrou em contato com o personagem e o feriu, e quando o personagem pegou uma estrela a tempo para recuperar vida.

# Escuta concorrente de multiplos canais

Isso é aplicado em `vida.go`, onde a função `admVida()`, que é chamada como uma rotina concorrente em `main.go` antes do inicio do loop principal de entrada, utiliza um `select { case }` ouvindo dois canais que recebem informações das rotinas de "spawn" de inimigos e de estrelas. A partir disso, caso o canal `heal_confirmation` receba true da rotina de estrelas e o jogador esteja com menos de 3 corações, um dos corações feridos é trocado por um coração cheio, permitindo que o jogador seja ferido mais uma vez. Entretanto, caso o canal `damage_confirmation` receba true da rotina de inimigos, que ocorre quando um inimigo atinge o jogador, um dos corações cheios é trocado por um coração ferido. Assim que todos os corações saúdaveis se tornarem corações feridos, a rotina passa true para o canal `gameOver`, que na próxima iteração do loop principal de entrada vai fazer com que o código caia em um break, finalizando o jogo.

# Comunicação com timeout

A comunicação com timeout é aplicada na rotina de estrelas. Caso o jogador ande por cima de uma estrela dentro do período de 10 segundos após seu nascimento, `personagemMover()` passa true para um canal compartilhado com `powerUpSPawnar()`, indicando a obtenção da estrela. Essa verificação é feita através de um `select { case }`, e possui um "case" para caso o jogador não consiga alcançar a estrela dentro do período de 10 segundos, removendo a estrela e imposibilitando o jogador de ser curado naquele momento.

# Relatório T2:

## Servidor RPC (autoridade do jogo)

Implementamos um servidor RPC em Go (arquivo `servidor/servidor.go`) usando `net/rpc` sobre TCP. Ele é a autoridade do estado compartilhado entre clientes, mantendo:
- Jogadores conectados (`players`), com id, nome, coordenadas (X,Y), direção e vidas.
- Um log sequencial de eventos (`events`) com um contador incremental (`Event.ID`).

Principais chamadas RPC expostas:
- `GameServer.Join`: registra um novo jogador, devolve seu próprio resumo, lista dos demais e o último `EventID` conhecido; gera um evento `player_join`.
- `GameServer.Update`: recebe “batimentos”/ações do cliente (posição atual, mudanças de vida e uma `Action` textual como `shoot`, `hit`, `heal`, `damaged`) e gera um evento `player_state` para os demais.
- `GameServer.Leave`: remove o jogador e gera `player_leave`.
- `GameServer.Poll`: devolve os eventos a partir de um `Since` informado (pull periódico pelos clientes).

Concorrência no servidor: todo acesso/alteração em `players` e `events` é protegido por `s.mu` (um `sync.Mutex`), garantindo consistência do estado e ordem dos eventos.

## Modificações no cliente (jogo)

Criamos `jogo/cliente_rpc.go` para um cliente RPC opcional, ativado quando a variável de ambiente `SERVER_ADDR` está definida.
- Ao iniciar, o cliente chama `Join` e recebe seu `ID` e o snapshot dos outros jogadores, desenhando-os no mapa como `SegundoJogador`.
- A cada 200ms, envia `Update` com posição atual (heartbeat) e faz `Poll` para aplicar os eventos remotos no mapa local (entradas, saídas e estados dos outros jogadores).
- Adicionamos um gancho global `rpcEmitAction(action string, lives int)` que, quando há conexão RPC, envia um `Update` com a ação. Esse gancho foi conectado em:
	- `personagemAtirar`: emite `shoot` ao iniciar e `hit` ao destruir um inimigo.
	- `vidaAdm`: emite `heal` ao recuperar e `damaged` ao tomar dano (inclui o total de vidas atualizado; se `lives=0`, o servidor também fica ciente da morte).

Importante: todo desenho/alteração do mapa no cliente continua respeitando o canal `lock` (token único) já usado pelo projeto.

## Garantia de execução única (exclusão mútua) e validação

No cliente (jogo):
- Mantivemos a estratégia de “token via canal” (`lock chan struct{}`) para garantir que apenas uma rotina por vez modifica o estado compartilhado (mapa, posição) e redesenha a tela. Qualquer trecho crítico faz `<-lock` antes e devolve o token com `lock <- struct{}{}` depois.
- O cliente RPC usa o mesmo `lock` sempre que escreve no mapa (por exemplo, ao desenhar/remover `SegundoJogador`), impedindo interleavings perigosos com o loop principal, tiros, inimigos e power-ups.

No servidor:
- Usamos `sync.Mutex` para garantir execução única nas seções críticas (cadastro/remoção de jogadores, registro de eventos e leituras consistentes), já que na parte do servidor o uso de mutex é permitido e adequado.

Validação prática:
- Rodamos múltiplos clientes conectados ao mesmo servidor e verificamos ausência de corrupção visual/estado (outros jogadores movem-se de forma fluida e limpa no mapa local).
- Execução com o detector de corrida do Go (opcional) ajuda a flagrar acessos simultâneos indevidos durante o desenvolvimento:
	- Servidor: `go run -race .` dentro de `servidor/`.
	- Jogo: `go run -race .` dentro de `jogo/` (o termbox funciona normalmente com `-race`).
- A ausência de avisos do `-race` e o comportamento estável ao interagir (mover, atirar, curar, sofrer dano) indicam que a exclusão mútua está efetiva e que os eventos RPC estão sendo aplicados sem condições de corrida.

 