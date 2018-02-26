/*
	no caso o master tem que ter uma lista de todos os nos para que ele possa saber quem entrou e
	quem saiu.
*/

/*
	qnd o master iniciar, vamos ter que gerar um contexto com uma funcao de cancel.
	se por acaso ele perder o lock, vamos chamar a funcao de cancel e as atividades do master enceram.
*/

/*
	criar mais contextos, tipo, um apra election, 1 para o cluster, etc...
	qnd for dar o stop, ir dando stop aos poucos nos contextos.
*/



/*
	1) toda vez que o scheduler iniciar ele tem que se cadastrar na lista de nodes.
	e vai ter que ficar fazendo um ttl para manter a linha ativa.

	2) ele tem que ficar tentando dar lock para ser o master, se ele conseguir dar o lock, ele sabe
	que vai ser o master. enquanto isso, todos ficam tentando e tem o ttl para manter o master com ele
*/

/*
	antes de tentar pegar o lock do mnaster, tem que se registarar como um nó...
*/


// vamos inserir e dps chamar o keep alive, mas se o objeto sair, o keep alive vai dar erro,
// nesse caso temos que tentar inserir mesmo.
// nesse caso, nao sei se a fn vai ser util.

/*
	tem que ter uma goroutine tb que fica varrendo os nodes procurando pela partition key. quem vai
	setar esse valor vai ser o node master, mas os outros nodes vao ter que respeitar esse valor.
*/
// func (c *Client) register() {
// 	defer func() {
// 		if err := recover(); err != nil {
// 			go c.register()
// 		}
// 	}()

// 	keepAlive, err := c.Register.Register(context.Background(), c.id, 1*time.Minute)
// 	if err != nil {
// 		panic(err)
// 	}

// 	for {
// 		if err := keepAlive(); err != nil {
// 			panic(err)
// 		}
// 		<-time.After(1 * time.Minute)
// 	}

// }


// func (c *Client) Start2() {
// 	ttl := 10 * time.Second
// 	keepAlive, err := c.Register.Register(context.Background(), uuid.NewV4().String(), ttl)
// 	if err != nil {
// 		panic(err)
// 	}

// 	for {
// 		if err := keepAlive(); err != nil {
// 			panic(err)
// 		}
// 		<-time.After(ttl)
// 	}

// 	// go func() {
// 	// 	chanConsumer, chanErr, err := c.ConsumerFetcher.Find(context.Background())
// 	// 	if err != nil {
// 	// 		// log, sleep and go for the next one.
// 	// 	}

// 	// 	for {
// 	// 		select {
// 	// 		case consumer := <-chanConsumer:
// 	// 			_ = consumer
// 	// 			// enviar para um dispatcher que vai ligar o processador.
// 	// 		case err := <-chanErr:
// 	// 			_ = err
// 	// 			// log, sleep and go for the next one.
// 	// 		}
// 	// 	}
// 	// }()
// }

// func (c *Client) startMasterElection() {
// 	var err error

// 	for {
// 		var locked bool

// 		if c.isMaster {
// 			locked, err = c.Locker.RefreshLock(context.Background(), "node.master", c.id, c.MasterElectionInterval)
// 			if err != nil {
// 				panic(err)
// 			}
// 		} else {
// 			locked, err = c.Locker.Lock(context.Background(), "node.master", c.id, c.MasterElectionInterval)
// 			if err != nil {
// 				panic(err)
// 			}
// 		}

// 		if locked {
// 			pretty.Println("parabens, voce eh o master....")
// 			// disparar as triggers que o master eh responsavel como regerar as particoes.
// 		}

// 		if c.isMaster {
// 			<-time.After(c.MasterElectionInterval / 2)
// 		} else {
// 			<-time.After(c.MasterElectionInterval)
// 		}
// 	}
// }
