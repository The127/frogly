-- +migrate Up

-- asked_at is microseconds since 1970 in UTC, like an idea's planted_at.
-- The unique pair also finds a project's requests, and the index on the
-- user the projects someone asked.
create table join_requests
(
    id         text    not null primary key,
    version    integer not null,
    project_id text    not null references projects (id),
    user_id    text    not null references users (id),
    asked_at   integer not null,
    unique (project_id, user_id)
);

create index join_requests_by_user on join_requests (user_id);

-- +migrate Down

drop table join_requests;
