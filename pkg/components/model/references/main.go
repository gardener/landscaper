package main

import (
	"fmt"
	"time"
)

func main() {
	nodeA := &NodeImpl{id: IDImpl("A")}
	nodeB := &NodeImpl{id: IDImpl("B")}
	nodeC := &NodeImpl{id: IDImpl("C")}
	nodeD := &NodeImpl{id: IDImpl("D")}
	nodeE := &NodeImpl{id: IDImpl("E")}
	nodeA.refs = []Reference{
		&ReferenceImpl{node: nodeB, dur: time.Second},
		&ReferenceImpl{node: nodeC, dur: 4 * time.Second},
	}
	nodeC.refs = []Reference{
		&ReferenceImpl{node: nodeD, dur: time.Second},
	}
	nodeB.refs = []Reference{
		&ReferenceImpl{node: nodeD, dur: time.Second, err: fmt.Errorf("failed to get D\n")},
	}
	nodeD.refs = []Reference{
		&ReferenceImpl{node: nodeE, dur: time.Second},
	}

	m := NewManager()
	err := m.manage(nodeA)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(m.tasks)
}
