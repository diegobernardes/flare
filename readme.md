---
GET    /consumers
GET    /consumers/:id
PUT    /consumers/:id
DELETE /consumers/:id
POST   /consumers

GET    /consumers/:consumer_id/producers
GET    /consumers/:consumer_id/producers/:id
PUT    /consumers/:consumer_id/producers/:id
DELETE /consumers/:consumer_id/producers/:id
POST   /consumers/:consumer_id/producers

---
stats vamos usar o que? prometheus? redis?
de alguma forma queria dizer o status dos consumers, tipo, qual node está prcessando.
quants req por seguindo ou mensagem por segundo, etc.. etc..

a mesma coisa nos producers, queria fazer a mesma coisa.
vamos fazer isso mais para frente.

---
o que queremos?

- election
quero eleger um master dentro dos nós.

- quero poder lockar chaves
e caso o nó responsável saia, quero que o lock seja liberado.
como que eu sei o que um nó esta processando? fazendo um range query e mantendo um estado no master.

---
oq podemos trocar? tipo, o banco e a fila é facil, o etcd, vai ser absurdamente dificil.

---
- registra no cluster (ok)
  - processar consumers
  - eleicao master (ok)
    - dispatch de consumers

se o primeiro falhar, todos embaixo tem que ser reiniciados.

---
aceitar o 'with replication' na configuracao.

---
pegar o status de um consumer.
outra coisa eh, poder pausar um consumer.

---
se inspirar um pouco nos endpoints de cluster do elasticsearch: https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster.html

---
para o vgo, fazer um linter que verifica o codigo, compara com a versao anterior e verifica se precisa fazer um bump na versao.
nada impede do cara quebrar dentro da propria versao, massss, ai ja eh outra historia.

---
{
  version: "",
  startTimeEpochSecs: ""
  currentTimeEpochSecs: "",
  uptime: "131232seconds"
}

version: <major>.<minor>.<commit#>.<git sha>.<date>.<time> 
how this play nice with semver?

/status
exibir quantidade de servidores conectados, status detalhados por servidor.
exibir quantidade de consumers sendo processados.

gerar as metricas no newrelic, prometheus, infrluxdb, etc...
https://github.com/go-kit/kit/tree/master/metrics
mas como fazer o error handling!? tipo do new relic, nesse caso teriamos que ter outra coisa.

---
profile em producao!

---
adicionar suporte para notificacao, qnd um consumer for criado, vamos notificar diretamente o master do cara, ele vai ver quem vai calcular e em seguida
vai notificar diretamente a pessoa resposnavel.

---
olhar o https://github.com/victorcoder/dkron
entender as configs, como o serf funciona, etc...

olhar tb os concorrentes dele.

---
em tese eu teria que varrer todo o codigo de tempos em tempos procurando por coisas que eu ja nao preciso processar.
tipo nos que sairam, etc....
pensar em um event log da vida.

o node de x em x tempos vai ter que varrer todos os consumers e pegar as ids.
se por acaso achar algo que nao esteja no estado, tem que mandar deletar.

isso precisa rodar pelo menos 1x qnd o cara inicia como master.
mas como fazer isso!? acho que nao vai ter como sem quebrar a abstracao...

como podemos monitorar os dados no cassandra?

criar outra tabela com consumer_id e node_id, o consumer qnd for rodar, gerar um lock mesmo assim.


de x em x tempos, um tempo maior. 
varrer os consumers e depois varrer os nodes e ver quem nao ta e falar que saiu.


o node no cassandra, de tempos em tempos vai varrer os leases e vai varrer tb os consumers.
dai ele vai pegar as ids e ver quem mudou pra passar pra frente.
espero que isso nao seja lento no futuro qnd tiver muitos consumers.
podemos colocar o discovery para uns 5 minutos, algo assim. nao precisa ser instantaneo.

---
ainda esta quebrada a logica. posso avisar que um node saiu, ok, isso vai funcionar.
mas nao vou garantir que os consumers vao ser associados para o cara certo.

posso gerar um log.
os nós consomem os logs, o master limpa os logs. mas qnd limpar os logs?

podemos criar uma tabela de associacao mesmo.


posso guardar junto com o lease. se o nó sair, automaticamente ele vai liberar o lock.


acho que vou fazer o ring mesmo, dai todo mundo fica rodando procurando oq processar de x em x tempos.

vamos colucar os assigns de consumer para node dentro de um array.


o cara vai se registrar.
dps vai tentar se eleger o master.
dps vai pegar os consumers e associar aos outros nodes.



podemos inverter, quem procura algo para processar é o node, se ele perceber que algo esta consumindo muito processo, ele pode largar o consumer.
logo, alguem vai pegar.

---
no producer, vamos definir um ttl de consumo.
podemos colocar o ttl maximo na config, mas vamos deixar o cara escolher.