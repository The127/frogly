-- +migrate Up

-- SQLite can't change a column's check, so joining moves to a new column
-- that also allows request, and keeps its name and values
alter table projects
    add column joining_new text not null default 'closed'
        check (joining_new in ('open', 'request', 'closed') and (joining_new = 'closed' or visibility = 'public'));

-- the version trigger only lets an update through that writes the next
-- version
update projects set joining_new = joining, version = version + 1;

alter table projects drop column joining;
alter table projects rename column joining_new to joining;

-- +migrate Down

alter table projects
    add column joining_old text not null default 'closed'
        check (joining_old in ('open', 'closed') and (joining_old = 'closed' or visibility = 'public'));

update projects
set joining_old = case joining when 'open' then 'open' else 'closed' end,
    version     = version + 1;

alter table projects drop column joining;
alter table projects rename column joining_old to joining;
