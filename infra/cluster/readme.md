/*
	de x em x tempos, pegar todos os nós no banco.
	verificar qual o estado anterior, se tiver.
	dai vamos saber quais nós sairam e quais entraram.

	na primeira vez, fazer um cursor sobre todos os consumers e ir lendo e gerando um estado local.
	depois processar essse estado para gerar as atribicoes para os nós.

	executar as atribuicoes. vamos usar a mesma tabela de lease.
	o scheduler gera o lease e o worker verifica se o lease esta ativo de tempos em tempos.
	se nao estiver mais, ele para de processar. o problema eh que vai ficar meio falho, ele que tem
	que gerar o lease.

	nodeID = 123
	consumerID = 456

	a idéia é manter tudo na tabela de lease, se der.


	entao o scheduler gera um lease para um nó processar um consumer.
	ele vai automaticamente ficar renovando o lease para gente.

	e se o scheduler gerar o lease e o worker manter o lease ativo? (melhor dos mundos)

	o scheduler vai gerar um lease de 1 minuto
	a cada 30 segundos o worker procura por novos consumers.
	qnd ele pegar o nome dele, ele vai comecar a dar renew no lease.

	podemos criar um tipo de lease diferente que compoe o primeiro e só chamamos o renew nele.

	mas podemos entrar em concorrencia aqui, oq acontece se o scheduler tirar o lease e em seguida
	o worker atualizar o lease? isso nao vai ser possivel por causa do 'IF'

	agora o segundo problema, o lease está para um worker, dai ele perde o lease para outro worker.
	o primeiro, nao vai coinseguir atualizar o lease, mas ele ainda tem tempo até chegar a atualizacao.
	nesse meio tempo, o segundo pode comecar a trabalhar, e nesse memento (mo), vamos ter concorrencia.

	temos que ter uma flag para indicar que ainda estamos processando ou algo do tipo.
	talvez implementar uma maquina de estados do lado do scheduler.

	se alguem ja estiver processando, ele envia um sinal dizendo que eh para parar de processar.
	qnd ele perceber que o nó não está mais processando, ele gera o estimulo novo colocando o
	consumer para o nó novo que vai então comecar a processar e a maquina de estados termina.

	podemos colcoar no lease um campo chamado release, dai o master pode mover para true e o worker
	se for true, ele não vai conseguir renovar. ( nao sei se isso eh possivel )

	podemos crira uma tabela de schedules. la colocamos os consumers e os nós que vao processar.
	os consumers por sua vez vão fazer um lock na lease para trabalhar. mas ai vamos entrar no mesmo
	problema.

	acho que nesse momento teriamos que ter uma comunicacao entre os nós. o scheduler vai chegar e
	falar: 'no x, processar '

	essa comunicacao vai ter que ser feita pelo nats. o scheduler vai falar diretamente com o no
	e pedir para ele comecar a processar e ele vai dar um lock na chave


	### best solution ################################################################################
	vamos criar uma tabela de key value, onde, vamos ter um consumer e um nó. quem vai gerar o dado lá
	vai ser o scheduler.
	o nó fica procurando nessa tabela com um filtro de data.
	se ele achar algo pra ele, ele tenta lockar e começa a processar.
	qnd ele for renovar o lock, ele vai nessa tabela denovo pela id e verifica se o cara ainda esta
	associado para ele, se estiver, ele continua, se nao, ele para de processar e o lock eventualmente
	vai ser liberado. então, o nó correto vai pegar e processar.

	acho que nao precisamos criar outra tabela, pode ser a de lease?

	registry, lock, assign

	/consumers/123 | 456 | assign | dt

*/

/*
	do lado do cassadnra, vamos ter que manter um estado la de quais consumers ele retornou e um
	context.Context daquele consumer.
	o proprio cassandra vai ficar pingando no banco de tempos em tempos para descobrir se o cara ainda
	esta associado. se nao estiver, vamos cancelar o contexto e deletar a chave.

	a chamada pro fetch tem qeu travar até termos algo para processar ou o contexto ser cancelado.

	na verdade preciso retornar um consumer e um contexto para cada consumer. pq eu posso querer parar
	eles individualmente, nao!?
*/

/*
	estavamos descrevendo o runner, mas esse cara eh o scheduler!

	precisamos de otura tabela alem da lease!?

	qnd for assign, nao podemos deletar da tabela! apenas tirar o node_id e atualizar o updated_at.
*/

/*
	bucar todos os consumers depois de uma certa data, gravar a data em um estado e sempre buscar
	acima dessa data.

	esse cara tb deve consumir a listagem dos lease de registry.

	dai ele junta os dois e forma o scheduler que envia os consumers para os nós.
*/




/*
	vamos chamar o fetcher que vai buscar um consumidor e vai retornar com um contexto. de x em x
	tempos o processo vai ter que ir no consumidor ver se ele ainda existe, se existir, nao faz nada.
	se nao existir, cancelar o contexto para parar o processamento. qnd atualiza, a data avanca, logo
	o processo de fetcher vai pegar denovo, se ele pegar algo que ja esta processando, ele primeiro da
	um stop e depois start.

	nesse meio tempo, o processo vai buscar e ficar buscando todos os nós. qnd houver alteracao, vamos
	rebalancer o cluster.

	esse processo vai pegar e atribuir os consumers para os nós.
*/

/*
	acho que vou criar um outro channel para ir processando enquanto consome geral
*/

/*
	mudar de channel para funcao com retorno de erro, se der erro, o cara vai retentar depois.
	quem eh responsavel por retentar!?
*/
