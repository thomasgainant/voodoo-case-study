package worker

// Worker processes all requests routed to it.
// Business logic is not yet implemented.
type Worker struct {
	id int
}

func New(id int) *Worker {
	return &Worker{id: id}
}

func (w *Worker) ID() int { return w.id }
