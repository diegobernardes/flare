index...

o worker pode ter uma api e essa api que vai ser usado para a comunicacao entre a api e o worker.
sounds good?

se for no mesmo processo, nao faz sentido ser http.. elixir faz sentido aqui


tema: https://github.com/tabler/tabler
https://themeforest.net/item/thesaas-responsive-bootstrap-saas-software-webapp-template/19778599?_ga=2.5807283.1633615957.1535052516-644329454.1533382661


qnd um subscription é criado, precisamos de uma task para poder criar a fila no rabbitmq.

durante o setup do subscription, algumas mensagens podeem ser perdidas, porque ele só entra na partition depois de tudo estar pronto.

o subscription pode ter um status. tipo, ativo, inativo ou criando?


no modo passive, temos que ter duas filas. uma fila para delivery.
essa fila vai estar sendo sempre processada como qualquer outra fila.
e a fila de enqueue. qnd o cliente pedir, por exemplo, 10 msgs, vamos nessa fila de enqueue e colocamos na fila de delivery
e todo processo continua normalmente.


quanto as filas, o ideal seria fazer um lock e somente 1 processo processa a fila para evitar problemas de sinfronizacao.
a sincronizacao pode ser local.

o problema é, e se a quantiadde de mensagens for fora muito alta para apenas 1 consumidor?
temos que ter um processo de rebalance tambem.

como resolver isso?

podemos ter a abordagem do kinesis e fazer sharding de filas. logo, poderiamos ter mais filas para o mesmo subscribe.
isso seria feito durante a criação do subscribe? configuração?


https://medium.com/opentracing/tracing-http-request-latency-in-go-with-opentracing-7cc1282a100a


não anotar todos os erros, só retornar, mas na origem, pegar o erro e gerar o stack 
no final abrir o erro de origem e pegar o stack para printar.
no log temos que guardar também o git commit hash porque assim referencia aplicação.


docker run --rm -d -p 27017:27017 --name flare-mongodb mongo:4.0.1-xenial
docker run --rm -d -p 5672:5672 -p 15672:15672 --name flare-rabbitmq rabbitmq:3.7.7-management


# 
como cuidar dos migrations?
fazer tipo o rails?

diferenca disso int32(2) para isso (int32)(2)


pensar em como fazer migrations de database.


// isso tem que ser no setup. vamos ter que ter essa chave.
// durante a conexão podemos ver se já iniciamos o banco para garantir.
criar uma tabela de marcacao para acompanhar os migrations, index, etc...


https://github.com/golangci/awesome-go-linters


https://github.com/apex/apex
take a look


como prevenir ataques de post muito grande, tipo, enviar 1tb nopost, como resolver isso?


se a api e o worker estiverem rodando no mesmo processo, eles se chamam pelo computador.
se nao, por http.
ou sempre fazemos deploy junto e a api tem conhecimento do pacote que faz o push pro rabbit? isso eh mais facil...


o mongodb já suporta transaction, vamos usar?
mas tem que pensar em algo genérico para isso...

https://docs.aws.amazon.com/sns/latest/dg/DeliveryPolicies.html#delivery-policy-pre-backoff-phase


https://medium.com/@benbjohnson/standard-package-layout-7cdbc8391fc1


https://arslan.io/2018/08/26/using-go-modules-with-vendor-support-on-travis-ci/