package example

import "fmt"

type Queries struct{}

func (q *Queries) InsertAuditLog(msg string) error {
	return fmt.Errorf("audit log: %s", msg)
}

func badDiscarded(q *Queries) {
	_ = q.InsertAuditLog("test") // want "InsertAuditLog error discarded"
}

func badSwallowedIfInit(q *Queries) {
	if err := q.InsertAuditLog("test"); err != nil { // want "InsertAuditLog error not returned"
		fmt.Println(err)
	}
}

func badSwallowedSeparate(q *Queries) {
	err := q.InsertAuditLog("test") // want "InsertAuditLog error not returned"
	if err != nil {
		fmt.Println(err)
	}
}

func goodReturned(q *Queries) error {
	if err := q.InsertAuditLog("test"); err != nil {
		return err
	}
	return nil
}

func goodReturnedSeparate(q *Queries) error {
	err := q.InsertAuditLog("test")
	if err != nil {
		return err
	}
	return nil
}

func goodDirectReturn(q *Queries) error {
	return q.InsertAuditLog("test")
}
