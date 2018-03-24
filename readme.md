mudar o consumer para o watch e o load receberem uma id.

---
oq acontece se o consumer assign mudar? tipo, um update.
nesse caso, nao vamos saber que o cara mudou.

posso analisar o prev kv, dai chamar a funcao 2x. com o delete.

---
e se a mensagem na fila (sqs) estiver errada? ficar retentando? 
configurar e obrigar a ter um redrive?
manter um cache interno e deletar a mensagem caso chegue em uma retentativa maxima!?

---
vamos suportar um sistema multi tenant e com utilizacao entre eles?

```
/consumers
/tenants/123/consumers
```

acho que a ideia eh ter somente 1 camada de tenant, na verdade, poderiamos chamar de account.
ou entao, nem precisa ter endpoint, pode ser header na chamada.

account-id: xyz

---
vale a pena usar o esquema que estamos fazendo de consumer/producer e abandonar o modelo antigo!?
temos que pensar nisso.

podemos esquecer esse esquema de REST e trabalhar em cima apenas das mensagens.

o bom do modelo novo eh que ele funciona com grpc, graphql, etc...

---
o payload nao eh obrigatorio. mas se ele nao existir, nao vamos conseguir fazer a questao do controle de entrega. vamos ser um proxy transparente.
tudo oq chega, vamos passar para frente.

na hora de criar o producer, nao vamos poder filtrar e ou controlar o que ja foi entregue.

{payload} -> {payload}

se eu nao conseguir entregar no destino, faco oq? mantenho ateh conseguir entregar
retentatvas, tempo, etc...?

se eu estiver na aws, o consumer pode ser um sns, dai eu vou registrando no sns os sqs que vao receber a mensagem. o msm vale pro rabbitmq.
na verdade se o consumer e o producer forem na aws, se sim, podemos fazer esse esquema do sns, nos outros casos, temos que ir na mao.

---
talvez o start tenha que receber um contexto, dai qnd eu cancelar, eu cancelo todos ao mesmo tempo.
eh meio que o stop, pq o stop cancela o contexto. dai eo processos ai.
a diferenca vai ser no start que recria o contexto denovo.