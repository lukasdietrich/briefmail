
-- +migrate Up

create table "mailboxes" (
	"id"   integer not null primary key autoincrement ,
	"hash" varchar not null
) ;

create table "mails" (
	"id"          varchar not null primary key ,
	"received_at" integer not null ,
	"deleted_at"  integer ,
	"return_path" varchar not null ,
	"size"        integer not null ,
	"attempt"     integer not null
) ;

create index "idx_mails_received_at"
	on "mails" ( "received_at" ) ;

create index "idx_mails_deleted_at"
	on "mails" ( "deleted_at" ) ;

create table "recipients" (
	"id"           integer not null primary key autoincrement ,
	"mail_id"      varchar not null ,
	"mailbox_id"   integer ,
	"forward_path" varchar not null ,
	"status"       integer not null ,

	foreign key ( "mail_id" ) references "mails" ( "id" ) on delete restrict ,
	foreign key ( "mailbox_id" ) references "mailboxes" ( "id" ) on delete restrict
) ;

create unique index "idx_recipients_unique_forward_path"
	on "recipients" ( "mail_id", "forward_path" ) ;

create index "idx_recipients_status"
	on "recipients" ( "status" ) ;

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

