-- +migrate Up

-- every project so far was only there for its members
alter table projects
    add column visibility text not null default 'private'
        check (visibility in ('public', 'private'));

-- everyone sees the list of public projects, so it mustn't read every
-- project to find them
create index projects_public on projects (name, id) where visibility = 'public';

-- +migrate Down

drop index projects_public;
alter table projects drop column visibility;
