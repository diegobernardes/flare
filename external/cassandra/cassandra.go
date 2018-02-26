package cassandra

import "github.com/diegobernardes/flare/infra/cluster"

func member(value string, values []string) bool {
	for _, x := range values {
		if x == value {
			return true
		}
	}
	return false
}

func memberClusterStatus(value string, status []cluster.ConsumerStatus) bool {
	ids := make([]string, len(status))
	for i, s := range status {
		ids[i] = s.ID
	}
	return member(value, ids)
}
