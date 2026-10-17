-- +migrate Up

-- every project so far was only there for its members
alter table projects
    add column visibility text not null default 'private'
        check (visibility in ('public', 'private'));

-- +migrate Down

alter table projects drop column visibility;
