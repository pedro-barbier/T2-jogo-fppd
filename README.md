Projeto feito por Marcelo Veiga Neves e Pedro Henrique Barbieri

## Jogo de Terminal em Go

# Como compilar

1. Instale o Go
2. Inicialize um novo módulo "jogo":

```bash
go mod init jogo
go get -u github.com/nsf/termbox-go
```

3. Compile o programa:

Linux:

```bash
go build -o jogo
```

Windows:

```bash
go build -o jogo.exe
```

Também é possivel compilar o projeto usando o comando `make` no Linux ou o script `build.bat` no Windows.

## Como executar

1. Certifique-se de ter o arquivo `mapa.txt` com um mapa válido.
2. Execute o programa no termimal ou pelo `jogo.exe` no Windows:

```bash
./jogo
```

# Requisitos mínimos de concorrência e sincronização aplicados:

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

