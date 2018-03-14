---
https://github.com/json-iterator/go
https://github.com/buger/jsonparser

---
formato do consumer

```json
{
  "id": "7a5001d3-665a-402a-a33b-3ad32febefa8",
  "source": {
    "type": "aws.kinesis",
    "stream": "some stream"
  },
  "payload": {
    "id": "id",
    "revision": {
      "field": "updatedAt",
      "format": "2006-01-02T15:04:05Z07:00"
    }
  },
  "createdAt": "2018-02-20T23:15:38-03:00"
}
```

```json
{
  "id": "7a5001d3-665a-402a-a33b-3ad32febefa8",
  "source": {
    "type": "aws.sqs",
    "arn": "some arn",
    "concurrency": 100
  },
  "payload": {
    "id": "id",
    "revision": {
      "field": "updatedAt",
      "format": "2006-01-02T15:04:05Z07:00"
    }
  },
  "createdAt": "2018-02-20T23:15:38-03:00"
}
```

---
separar o source do consumer/producer? se sim, vai ficar mais facil colocar configuracoes globais como rate limit.

```json
{
  "id": "7a5001d3-665a-402a-a33b-3ad32febefa8",
  "type": "aws.sqs",
  "arn": "some arn",
  "credentials": {
    "region": "",
    "account": "",
    "secret": ""
  },
  "rateLimit": {
    "consumer": {},
    "producer": {},
    "concurrency": 100
  },
  "createdAt": "2018-02-20T23:15:38-03:00"
}
```

vamos ter endpoints assim: 
/sources
/sources/:id
/sources/:id/status (diz como estamos nos rate limits, e o status de conexao caso eles tenham)

/consumers
/consumers/:id
/consumers/:id/status (diz se esta consumindo ou nao e qual maquina esta consumindo, ou quais maquinas)

/producers
/producers/:id
/producers/:id/status (msm coisa do consumer)

/nodes
/nodes/:id

no nodes vamos guardar informacoes do cluster atual. incluindo informacoes de onde estamos rodando, aws, region, etc.. etc..
o scheduler tem que ser o mais fair possivel.

---
no scheduler vamos ter uma abordagem como o kubernetes e cassandra, suportar multi cluster, etc.. etc..?
se sim, ele vai fazer o schedule pelo q? consumer? producer? quem fica em qual lugar? podemos dar uma preferencia? setar? escolher?

---
como estamos criando consumers, acredito que la seja o lugar ideal para ter as implementacoes.
entao, o consumer tem que entender quem ele es ta acessando? se sim, de certa forma vai ficar mais facil fazer o parse.

---
se por acaso um dos consumers nao ligar?
acho que temos que ter uma interface de status para exibir uma informacao do cluster.


/status (mostra qnts consumers estao sendo processados, mensagens, goroutines, etc...)

---
vou inicializar os providers independentemente, eles vao ter uma implementacao propria do repositorio.
eles vao se inicializar soiznhos e vamos passar a funcao do consumer para eles para que possamos colocar na fila de processamento.


---
futuramente vamos ter que ter um scheduler, ele que vai determinar quem vai proecsar em qual lugar.
no final acho que fica a mesma coisa, mas a interface vai ter um man in the middle que vai controlar oq cada um recebe.

---
talvez não tenhamos mais o worker de partition, somente spread e delivery.

---
implementar as regras de retry do kinesis. existem os limites da api (get shard iterator por exemplo)

---
em caso de um worker nao subir, vamos ter que retentar em um determinado momento.
quem iniciar os workders, no caso um scheduler, em caso de falha temporaria, deve agendar o start depois de 1 minuto. 
mas se for um erro grave, tipo, a fila foi deletada, algo que nao tem como remediar, nesse caso o worker nao vai subir.

---
temos que ter um painel com o status dos workers. talvez algo como '/status'
o ideal é mostrar qnts maquinas estao processando, qnts goroutiones, metricas, etc...

---
permitir acoes especificas nos sourecs, como no caso do kinesis, posso mandar deletar/alterar o offset.

---
a aws tem uma serie de limites globais, acho que podemos de alguma forma criar um tracking disso. e tem outro porem, algumas retricoes
sao por conta, outras por recurso. internamente, deveriamos tracker tudo isso e evitar fazer as chamadas se estiver claro que vai dar problema.
isso nao nos livra do problema, visto que, alguem pode chamar por fora o recurso, mas atenua.

---
vamos ter que adicionar a dependencia de um sistema de discovery, vamos deixar abstraido, seja la o etcd ou consul.

---
vamos ter que trabalhar no modelo master/slave. no nosso caso, o master vai ser o scheduler, ele é responsável por distribuir os jobs entre as maquinas.
as maquinas podem ter configs diferentes, no caso, master only ou master elegible.

ou entao podemos adotar um padrao tipo o cassandra, onde todos os nós são iguais. cada nó vai ter que guarar o estado dos outros nós.
qnd um worker for iniciar, vamos pegar a id e passar por um MPH, ele que vai dizer quem controla aquele worker.
no caso de um nó entrar ou sair, vamos recalcular os hashs.
como cada nó sabe tudo dos outros nós, vamos conseguir dessa forma pedir para alguma máquina executar o processo.
essa parte vai ter que ser sincronizada para evitar sobrecarregar a máquina com menor carga, visto que todos vão querer enviar para ela.

esse estado, podemos gravar em memória, e caso seja um problema, passar para um banco como o badger.

---
somente o próprio nó vai poder controlar a informação dele no cluster, os outros vão ser read only.
nesse caso, qnd formos subir um job em um nó, vamos mandar um request diretamente para ele, ele vai dizer se vai aceitar ou não a requisição.
se aceitar, ele vai atualizar o estado dele e notificar os outros.

---
definir um delay na entrada ou saida de nós do cluster.
se um nó perceber que saiu do cluster, dependendo doq estiver fazendo (kinesis), ele tem que parar de processar até voltar para o cluster.
isso em tese deveria ser uma chave de configuração.
vamos ter que ter um valor de split brain para poder determinar quem eh o cluster no caso de uma falha.

---
os rate limits ainda vão ter que estar em um banco.
o service discovery vai ser feito tipo o elasticsarch, vamos implementar para as clouds e vamos suportar o unicast.

---
o scheduler vai conhecer as implementacoes!? nesse caso ele teria que conhecer o kinesis.
para poder pegar os shards e entao escolher quem vai processar.

---
oq da pra fazer eh seguir a sugestao do yuri. criar um processo para o flare que soh vai entender a ideia de uma mensagem.
ele sabe pegar uma mensagem, gerar a diferenca e enviar para um producer.

os consumers e producers estariam em processos separados e poderiam escalar de maneira independente.
o kinesis teria sincronizacao, o sqs não.

a comunicação entre os consumers e os producers seria como?

---
scheduler!!!

// esse cara vai receber os requests do consumer.
// ele vai escolher quem vai processar oq.
// ele vai guardar o estado do kinesis e persitir em um banco.

// oq acontece se um stream parar?

// esse cli ent tem que ter um metodo que o consumer via chamar qnd algo for criado, ele que vai
// cuidar da iniailizacao.
// ele vai tb ter acesso ao banco.

/*
	o scheduler vai trabalhar como se fosse um supervisor do erlang.
	vamos procurar novos consumers para processar, se acharmos, vamos enviar para um dispatcher.

	se o cara for um kinesis, ele vai disparar outro loop para monitorar os shards e vai fazer a msm
	coisa. inclusive ele vai controlar se o shard morreu ou nao.
*/


// switch *streams.StreamDescription.StreamStatus {
// case kinesis.StreamStatusCreating:
// 	// should wait a little to process, maybe retry in x minutes.
// case kinesis.StreamStatusDeleting:
// 	// should stop the worker and mark it as never gonna work again.
// case kinesis.StreamStatusActive, kinesis.StreamStatusUpdating:
// 	// proceed
// }

// aqui tem que ter uma forma de persistencia tb que vai cuirar do estado do kinesis, gravar o
// offset de cada shard, etc...


---
cria um lock no cassandra.
quem pegar o lock vira o master.
esse nó então vai ter que ficar atualizando o lock para não perder o master.

dai ele vai na listagem de nós e atribui uma id para cada um.
a partir dai, os nos qnd pegarem uma id, podem comecar a processar os consumidores.
vao procurar todos os consumidores que estiverem associados para a id dele e sem estado. (que vai ser um ttl)
a cada x tempos, vamos bater no banco para pegar novos consumidores para processar.

de x em x tempos, vamos ter que ficar tentando conferir a id pra ver se mudou, se mudou, temos que parar tudo e repetir os passos de cima.

se colocarmos o ttl da chave de 1 minuto, garantimos que em ateh 1 minuto tudo vai estar processando, distribuido em todos os nós.

futuramente, poderiamos conversar pelo nats. confinuar fazendo os locks no banco, mas comunicar via nats.
ou entao usar o nats + raft e sair do banco, que seria o melhor estado...

---
suportar update no resource, o scheduler vai ter que ver o tempo e ver se o cara mudou, se sim, recarrega.

---
isso pode ser um problema!
http://datanerds.io/post/cassandra-no-row-consistency/
https://news.ycombinator.com/item?id=13033243

---
criar o endpoint '/consumers/:id/status'. Esse endpoint vai mostrar os nós que estão processando.
um consumer pode ser processado por mais de uma maquina. exemplo, as maquinas vao ter uma config de max quantity of goroutines.
nesse caso, o scheduler pode direcionar um procesasmento para uma maquina, desde que ela tenha capacidade para isso, ou entao, se perceber que nao tem capacidade.
pode quebrar a tarefa em pecados menores e mandar para maquinas separadas. 

isso vale para todos os providers, kinesis (shards), sqs, etc..

---
pro cassandra o update e o insert sao a msm coisa, e ai? isso influencia no codigo? lock and refresh lock?

---
adicionar opcao de pausar os consumers, um ou todos.

---
futuramente adicionar o nats para enviar diretamente para a pessoa a alteracao ao inves de ficar dependendo de loops sobre intervalos para pegar as alteracoes.

---
sincronização de datas entre os nós é muito importante.

---
como o cassandra eh colunar, podemos guardar os dados dos shards do kinesis nas colunas dele.

---
ler melhor sobre o lock: http://antirez.com/news/101

---
ao inves de ficar fazendo pooling na base, poderia trabalhar com um esquema de logs, onde eu tenhoum offset, e se estiver fora, soh pego os logs que andaram.
meio como o kafka funciona hoje.

---
ler sobre o chubby
http://research.google.com/archive/chubby.html

---
hoje estou usando o consensus que o cassandra suporta: https://www.datastax.com/dev/blog/consensus-on-cassandra

---
no lock, mudar a consistencia para serial.

---
por causa do update no consumer, os runners precisam de tempos em tempos ir no banco ver se o cara mudou.
se sim, tem que recarregar.

---
da forma que criamos o banco, vai ser facil expor as estatisticas de cada um.

---
só deixar deletar um consumer se nao tiver producer associado a ele.

podemos ser eventualmente consistente? ou precisamos ser sempre consistentes?

---
problema com banco eventualmente consistente? vamos perder mensagens.
vamos sempre ter que trabalhar no modo consistente do cassandra.

---
talvez seja interesssante sair do cassandra e ir para o mongodb.
na verdade, preciso de um banco de dados que tenha consistencia de dados.

---
expor mais configuracoes: repository, lock, consensus, cluster (hj, todos estao rodando no cassandra, mas futuramente, poderiamos mudar isso)

---
tipos de provider:

storage
election
consensus
kv

---
hoje temos o createdAt, temos que avancar o createdAt para pegar os consumers que estao mudando.
sendo que temos que ter 2 createadats, 1 para as alteracoes de usuario e um para lateracao de processamento.