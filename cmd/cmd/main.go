package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/kr/pretty"
)

func main() {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	pretty.Println(cli.MemberList(context.Background()))

	rch := cli.Watch(context.Background(), "/node", clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			pretty.Println(ev)
			fmt.Printf("%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)

			pretty.Println(cli.KV.Get(context.Background(), string(ev.Kv.Key)))
		}
	}
	// PUT
}
