package migrations

import (
	"context"
)

func migration01(ctx context.Context, db DBInterface) error {
	_, err := db.Exec(
		ctx,
		`
create sequence users_id_seq;

create type order_status_enum as enum ('REGISTERED', 'PROCESSED', 'INVALID', 'PROCESSING');

create table if not exists "user"
(
   id        bigint         default nextval('users_id_seq'::regclass) not null
       constraint users_pk primary key,
   login     varchar(255),
   password  varchar(255),
   is_active boolean,
   balance   numeric(10, 2) default 0,
   withdrawn numeric(10, 2) default 0
);

alter sequence users_id_seq owned by "user".id;

create unique index if not exists users_id_uindex
   on "user" (id);

create unique index if not exists users_login_uindex
   on "user" (login);

create table if not exists user_session
(
   user_id    bigint                   not null
       constraint user_session_user_id_fk
           references "user"
           on update cascade on delete cascade,
   token      varchar(64)              not null,
   created_at timestamp with time zone not null,
   expires_at timestamp with time zone,
   constraint user_session_pk
       primary key (user_id, token)
);

create index if not exists user_session_token_user_id_index
   on user_session (token, user_id);

create table if not exists withdrawal
(
   id           bigint         not null
       constraint withdrawal_pk
           primary key,
   "order"      varchar(255),
   sum          numeric(10, 2) not null,
   processed_at timestamp with time zone default now(),
   user_id      bigint
       constraint withdrawal_user_id_fk
           references "user"
           on update restrict on delete restrict
);

create table if not exists "order"
(
   "order"      varchar(255)                                              not null
       constraint order_pk
           primary key,
   user_id      bigint                                                    not null
       constraint order_users_id_fk
           references "user"
           on update restrict on delete restrict,
   status       order_status_enum default 'REGISTERED'::order_status_enum not null,
   accrual      numeric(10, 2),
   processed_at timestamp with time zone,
   uploaded_at  timestamp with time zone                                  not null
);

create index if not exists order_user_id_processed_at_index
   on "order" (user_id asc, processed_at desc);

create index if not exists order_uploaded_at_index
   on "order" (uploaded_at);

INSERT INTO revision VALUES(1);  
`)
	return err
}
