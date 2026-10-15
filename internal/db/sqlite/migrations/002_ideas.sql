-- +migrate Up

-- planted_at is microseconds since 1970 in UTC, since SQLite has no type
-- for time, and a number sorts the way time does
create table ideas
(
    id         text    not null primary key,
    version    integer not null,
    project_id text    not null references projects (id),
    planter_id text    not null references users (id),
    text       text    not null,
    planted_at integer not null
);

create index ideas_by_project on ideas (project_id, planted_at desc, id desc);

-- +migrate Down

drop table ideas;
