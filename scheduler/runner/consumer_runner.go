package runner

/*
	esse cara fica varrendo a colecao procurando por coisas novas para processar ou parar de processar.
	ele tem um mapa em memoria com o que ele esta processando.

	acho que seria interessante ter um ttl na coluna de processamento.
*/

type Client struct{}

func (c *Client) Start() {}

func (c *Client) Stop() {}
