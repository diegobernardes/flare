
/*
	pensar em algo como revisao nas alteracoes? pq se tivermos isso, podemos ir processando direto
	sem locks.


	por exemplo, podemos criar um campo de revisao, o etcd tem revisao, outros bancos poderiam mandar
	um campo de data como unix time. o problema que ainda sim, seria passivel de erro.
*/

/*
	como podemos colocar o scheduler para funcionar.

	1 - ele vai buscar todos os nós ativos e ao mesmo tempo ele vai dar um watch para escutar as
	alteracoes (entrada e saida de nós.).

	2 - vai fazer a mesma coisa com os consumers.


	a partir dai ele vai tomar a decisao
*/

/*
	implementar um mvvc.

	tipo, qnd algo for acontecer, vamos lockar o estado, entao ele nao muda mais, dai vamos fazer uma
	coipia do  estado e fazer a alteracao que desejamos, durante esse tempo, atualizacoes podem estar
	acontecendo, elas tem que ir par aum changelog. aplicamos o estado, mudamos o atual para o antigo
	e repetimos com o changelog até ele zerar.
*/

/*
	temos que montar o estado com o load e o watch eh para monitorar. carregamos os estados do node
	e do consumer pelo load, dai juntamos tudo e geramos o estado, depois disso, vamos processar o que
	vier pelo watch.
*/

/*
	vamos fazer um cache do lado do etcd e carregar geral em memoria, o watch, sempre antes de enviar
	algo, vai verificar no estado se ja tem algo la, se tiver, ele vai deletar.
*/


	// agora que eu tenho os ids e os updates, realizar as atualizacoes.
	// de alguma forma, precisamos travar os udpates ateh executar o estado que temos.
	// talvez gerar uma imagem e aplicar, e dps pra cada update, gferar uma revisao e aplicar novamente.
	// se nao conseguir executar algo, oq fazer? parar e sair!?
