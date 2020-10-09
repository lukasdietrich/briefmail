// Copyright (C) 2020  Lukas Dietrich <lukas@lukasdietrich.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package shell

import (
	"errors"
	"fmt"
	"time"

	"github.com/ktr0731/go-fuzzyfinder"

	"github.com/lukasdietrich/briefmail/internal/crypto"
	"github.com/lukasdietrich/briefmail/internal/mails"
	"github.com/lukasdietrich/briefmail/internal/storage"
	"github.com/lukasdietrich/briefmail/internal/storage/queries"
)

var (
	errNoDomains   = errors.New("there are no domains configured")
	errNoMailboxes = errors.New("there are no mailboxes configured")
	errNoAddresses = errors.New("there are no addresses configured")
)

func addDomain(ctx *cmdContext) error {
	domainName, err := ctx.ask("Domain name: ")
	if err != nil {
		return err
	}

	domainName, err = mails.DomainToUnicode(domainName)
	if err != nil {
		return fmt.Errorf("could not normalize domain %q: %w", domainName, err)
	}

	domain := storage.Domain{
		Name: domainName,
	}

	if err := queries.InsertDomain(ctx.tx, &domain); err != nil {
		return fmt.Errorf("could not store new domain %q: %w", domainName, err)
	}

	ctx.info("Domain %q added with id=%d.", domainName, domain.ID)
	return nil
}

func deleteDomain(ctx *cmdContext) error {
	domain, err := selectOneDomain(ctx)
	if err != nil {
		return err
	}

	if err := queries.DeleteDomain(ctx.tx, domain); err != nil {
		return fmt.Errorf("could not delete domain %q: %w", domain.Name, err)
	}

	ctx.info("Domain %q deleted.", domain.Name)
	return nil
}

func replaceDomain(ctx *cmdContext) error {
	domain, err := selectOneDomain(ctx)
	if err != nil {
		return err
	}

	newName, err := ctx.askWithDefault("New domain name: ", domain.Name)
	if err != nil {
		return err
	}

	newName, err = mails.DomainToUnicode(newName)
	if err != nil {
		return fmt.Errorf("could not normalize domain %q: %w", newName, err)
	}

	oldName := domain.Name
	domain.Name = newName

	if err := queries.UpdateDomain(ctx.tx, domain); err != nil {
		return fmt.Errorf("could not replace domain %q with %q: %w", oldName, newName, err)
	}

	ctx.info("Replaced domain %q with %q.", oldName, newName)
	return nil
}

func infoMailbox(ctx *cmdContext) error {
	mailbox, err := selectOneMailbox(ctx)
	if err != nil {
		return err
	}

	addresses, err := queries.FindAddressesByMailbox(ctx.tx, mailbox)
	if err != nil {
		return err
	}

	ctx.info("ID:   %d", mailbox.ID)
	ctx.info("Name: %q", mailbox.DisplayName)
	ctx.info("")
	ctx.info("(%d) Addresses", len(addresses))

	for _, address := range addresses {
		ctx.info("  %s@%s", address.LocalPart, address.DomainName)
	}

	return nil
}

func addMailbox(ctx *cmdContext) error {
	displayName, err := ctx.ask("Display name: ")
	if err != nil {
		return err
	}

	mailbox := storage.Mailbox{
		DisplayName: displayName,
	}

	password, err := ctx.password("Password: ")
	if err != nil {
		return err
	}

	if err := queries.InsertMailbox(ctx.tx, &mailbox); err != nil {
		return err
	}

	credentials := storage.MailboxCredentials{
		MailboxID: mailbox.ID,
		UpdatedAt: time.Now().Unix(),
	}

	if err := crypto.Hash(&credentials, password); err != nil {
		return err
	}

	if err := queries.UpsertMailboxCredentials(ctx.tx, &credentials); err != nil {
		return err
	}

	ctx.info("Mailbox added with id=%d.", mailbox.ID)
	return nil
}

func deleteMailbox(ctx *cmdContext) error {
	mailbox, err := selectOneMailbox(ctx)
	if err != nil {
		return err
	}

	if err := queries.DeleteMailbox(ctx.tx, mailbox); err != nil {
		return fmt.Errorf("could not delete mailbox %q: %w", mailbox.DisplayName, err)
	}

	ctx.info("Mailbox %q deleted.", mailbox.DisplayName)
	return nil
}

func passwdMailbox(ctx *cmdContext) error {
	mailbox, err := selectOneMailbox(ctx)
	if err != nil {
		return err
	}

	newPassword, err := ctx.password("New password: ")
	if err != nil {
		return err
	}

	credentials := storage.MailboxCredentials{
		MailboxID: mailbox.ID,
		UpdatedAt: time.Now().Unix(),
	}

	if err := crypto.Hash(&credentials, newPassword); err != nil {
		return err
	}

	if err := queries.UpsertMailboxCredentials(ctx.tx, &credentials); err != nil {
		return err
	}

	ctx.info("Password for mailbox %q changed.", mailbox.DisplayName)
	return nil
}

func addAddress(ctx *cmdContext) error {
	localPart, err := ctx.ask("Local-part [local-part@domain]: ")
	if err != nil {
		return err
	}

	localPart = mails.NormalizeLocalPart(localPart)

	domains, err := selectMultipleDomain(ctx)
	if err != nil {
		return err
	}

	mailbox, err := selectOneMailbox(ctx)
	if err != nil {
		return err
	}

	for _, domain := range domains {
		address := storage.Address{
			LocalPart: localPart,
			DomainID:  domain.ID,
			MailboxID: mailbox.ID,
		}

		if err := queries.InsertAddress(ctx.tx, &address); err != nil {
			return fmt.Errorf("could not store new address \"%s@%s\": %w",
				address.LocalPart, domain.Name, err)
		}

		ctx.info("Address \"%s@%s\" with id=%d added to mailbox %q.",
			address.LocalPart,
			domain.Name,
			address.ID,
			mailbox.DisplayName)
	}

	return nil
}

func deleteAddress(ctx *cmdContext) error {
	addresses, err := selectMultipleAddresses(ctx)
	if err != nil {
		return err
	}

	for _, address := range addresses {
		if err := queries.DeleteAddress(ctx.tx, &address.Address); err != nil {
			return fmt.Errorf("could not delete address \"%s@%s\": %w",
				address.LocalPart, address.DomainName, err)
		}

		ctx.info("Address \"%s@%s\" deleted.", address.LocalPart, address.DomainName)
	}
	return nil
}

func selectOneDomain(ctx *cmdContext) (*storage.Domain, error) {
	domains, err := queries.FindDomains(ctx.tx)
	if err != nil {
		return nil, err
	}

	if len(domains) == 0 {
		return nil, errNoDomains
	}

	index, err := fuzzyfinder.Find(domains, mapDomainSearch(domains))
	if err != nil {
		return nil, err
	}

	return &domains[index], nil
}

func selectMultipleDomain(ctx *cmdContext) ([]storage.Domain, error) {
	domains, err := queries.FindDomains(ctx.tx)
	if err != nil {
		return nil, err
	}

	if len(domains) == 0 {
		return nil, errNoDomains
	}

	indices, err := fuzzyfinder.FindMulti(domains, mapDomainSearch(domains))
	if err != nil {
		return nil, err
	}

	selectedDomains := make([]storage.Domain, len(indices))
	for i, index := range indices {
		selectedDomains[i] = domains[index]
	}

	return selectedDomains, nil
}

func selectOneMailbox(ctx *cmdContext) (*storage.Mailbox, error) {
	mailboxes, err := queries.FindMailboxes(ctx.tx)
	if err != nil {
		return nil, err
	}

	if len(mailboxes) == 0 {
		return nil, errNoMailboxes
	}

	index, err := fuzzyfinder.Find(mailboxes, mapMailboxSearch(mailboxes))
	if err != nil {
		return nil, err
	}

	return &mailboxes[index], nil
}

func selectMultipleAddresses(ctx *cmdContext) ([]queries.AddressWithDomain, error) {
	addresses, err := queries.FindAddresses(ctx.tx)
	if err != nil {
		return nil, err
	}

	if len(addresses) == 0 {
		return nil, errNoAddresses
	}

	indices, err := fuzzyfinder.FindMulti(addresses, mapAddressSearch(addresses))
	if err != nil {
		return nil, err
	}

	selectedAddresses := make([]queries.AddressWithDomain, len(indices))
	for i, index := range indices {
		selectedAddresses[i] = addresses[index]
	}

	return selectedAddresses, nil
}

func mapDomainSearch(domains []storage.Domain) func(int) string {
	return func(i int) string {
		return domains[i].Name
	}
}

func mapMailboxSearch(mailboxes []storage.Mailbox) func(int) string {
	return func(i int) string {
		return mailboxes[i].DisplayName
	}
}

func mapAddressSearch(addresses []queries.AddressWithDomain) func(int) string {
	return func(i int) string {
		address := addresses[i]
		return address.LocalPart + "@" + address.DomainName
	}
}
