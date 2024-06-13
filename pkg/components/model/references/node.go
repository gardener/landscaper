package main

type ID interface {
	String() string
}

type Reference interface {
	GetID() ID
	GetNode() Result
}

type Node interface {
	GetID() ID
	GetReferences() []Reference
}

type Result struct {
	node Node
	err  error
}

const (
	statusWaiting = "waiting"
	statusRunning = "running"
	statusDone    = "done"
)

type Task struct {
	reference Reference
	node      Node
	status    string
}

type Manager struct {
	tasks   map[ID]*Task
	workers chan bool
	results chan Result
}

func NewManager() *Manager {
	return &Manager{
		tasks:   make(map[ID]*Task),
		workers: make(chan bool, 5),
		results: make(chan Result, 1),
	}
}

func (m *Manager) manage(rootNode Node) error {
	m.tasks[rootNode.GetID()] = &Task{
		status: statusRunning,
	}

	m.results <- Result{
		node: rootNode,
	}

	for {
		select {
		case result := <-m.results:
			if result.err != nil {
				close(m.results)
				close(m.workers)
				return result.err
			}

			node := result.node
			t := m.tasks[node.GetID()]
			if t.status != statusDone {
				t.status = statusDone
				t.node = node

				refs := node.GetReferences()
				for i := range refs {
					if _, ok := m.tasks[refs[i].GetID()]; !ok {
						m.tasks[refs[i].GetID()] = &Task{
							status:    statusWaiting,
							reference: refs[i],
						}
					}
				}
			}

		case m.workers <- true:
			t := m.nextWaitingTask()
			if t != nil {
				t.status = statusRunning
				go m.handleTask(t)
			}
		}

		if m.allTasksDone() {
			close(m.results)
			close(m.workers)
			return nil
		}
	}
}

func (m *Manager) handleTask(t *Task) {
	defer func() {
		<-m.workers
	}()

	node := t.reference.GetNode()
	m.results <- node
}

func (m *Manager) nextWaitingTask() *Task {
	for _, task := range m.tasks {
		if task.status == statusWaiting {
			return task
		}
	}
	return nil
}

func (m *Manager) allTasksDone() bool {
	for _, t := range m.tasks {
		if t.status != statusDone {
			return false
		}
	}
	return true
}
