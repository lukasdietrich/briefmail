
-- +migrate Up

create table "mailboxes" (
	"id"   integer not null primary key autoincrement ,
	"hash" varchar not null
) ;

create table "mails" (
	"id"          varchar not null primary key ,
	"received_at" integer not null ,
	"return_path" varchar not null ,
	"size"        integer not null
) ;

create table "mailbox_entries" (
	"mailbox_id" integer not null ,
	"mail_id"    varchar not null ,

	primary key ( "mailbox_id", "mail_id" ),
	foreign key ( "mailbox_id" ) references "mailboxes" ( "id" ) on delete restrict ,
	foreign key ( "mail_id" ) references "mails" ( "id" ) on delete restrict
) ;

create table "domains" (
	"id"   integer not null primary key autoincrement ,
	"name" varchar not null
) ;

create unique index "idx_domains_unique_name"
	on "domains" ( "name" ) ;

create table "addresses" (
	"id"         integer not null primary key autoincrement ,
	"local_part" varchar not null ,
	"domain_id"  integer not null ,
	"mailbox_id" integer not null ,

	foreign key ( "domain_id" ) references "domains" ( "id" ) on delete restrict ,
	foreign key ( "mailbox_id" ) references "mailboxes" ( "id" ) on delete restrict
) ;

create unique index "idx_addresses_unique"
	on "addresses" ( "local_part", "domain_id" ) ;

