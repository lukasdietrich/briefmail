package queries

import (
	"github.com/lukasdietrich/briefmail/internal/storage"
)

// UpsertMailboxCredentials inserts new mailbox credentials. When there already is an entry for a
// mailbox, the row will be updated instead.
func UpsertMailboxCredentials(tx *storage.Tx, creds *storage.MailboxCredentials) error {
	const query = `
		insert or replace into "mailbox_credentials" (
			"mailbox_id" ,
			"updated_at" ,
			"hash"
		) values (
			:mailbox_id ,
			:updated_at ,
			:hash
		) ;
	`

	_, err := tx.NamedExec(query, creds)
	return err
}

// FindMailboxCredentials returns the credentials for a mailbox.
func FindMailboxCredentials(tx *storage.Tx, mailbox *storage.Mailbox) (*storage.MailboxCredentials, error) {
	const query = `
		select *
		from "mailbox_credentials"
		where "mailbox_id" = $1 ;
	`

	var creds storage.MailboxCredentials
	if err := tx.Get(&creds, query, mailbox.ID); err != nil {
		return nil, err
	}

	return &creds, nil
}
