package main

import (
	"fmt"
	"time"
)

type IDImpl string

func (id IDImpl) String() string {
	return string(id)
}

type NodeImpl struct {
	id   ID
	refs []Reference
}

var _ Node = &NodeImpl{}

func (n *NodeImpl) GetID() ID {
	return n.id
}

func (n *NodeImpl) GetReferences() []Reference {
	return n.refs
}

func (n *NodeImpl) String() string {
	return n.id.String()
}

type ReferenceImpl struct {
	node Node
	err  error
	dur  time.Duration
}

func (r *ReferenceImpl) GetID() ID {
	return r.node.GetID()
}

func (r *ReferenceImpl) GetNode() Result {
	fmt.Printf("starting %v at %v\n", r.GetID(), time.Now())
	time.Sleep(r.dur)
	fmt.Printf("finished %v at %v\n", r.GetID(), time.Now())
	if r.err != nil {
		return Result{err: r.err}
	}
	return Result{node: r.node}
}
